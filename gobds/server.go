package gobds

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// ServerConfig ...
type ServerConfig struct {
	Name          string
	LocalAddress  string
	RemoteAddress string
	// Transport is the upstream (BDS) transport: ""/"raknet" (default) or "nethernet".
	Transport string
	// NetherNet configures the client-facing NetherNet listener (in addition to RakNet).
	NetherNet NetherNetConfig
	// MOTD is the name shown in the server list. Falls back to Name when empty.
	MOTD string
	// MaxPlayers is the player capacity advertised in the server list.
	MaxPlayers   int
	ClaimService struct {
		Enabled bool
		URL     string
		Key     string
	}
}

// NetherNetConfig is the client-facing NetherNet (WebRTC) listener.
type NetherNetConfig struct {
	Enabled bool
	// UDPPort is the ICE UDP mux port. Required when Enabled.
	UDPPort int
	// PublicIP is an optional host ICE candidate (NAT 1:1).
	PublicIP string
}

func (s ServerConfig) validate() error {
	switch s.Transport {
	case "", "raknet", "nethernet":
	default:
		return fmt.Errorf("unknown transport %q", s.Transport)
	}
	if s.NetherNet.Enabled && s.NetherNet.UDPPort <= 0 {
		return fmt.Errorf("nethernet UDPPort required when enabled")
	}
	return nil
}

// Server represents a single server instance with its own Listener, DialerFunc, and minecraft.ServerStatusProvider.
type Server struct {
	Name          string
	LocalAddress  string
	RemoteAddress string

	// ClaimFactory is shared across all sessions on this server because claims are world-state
	// fetched periodically from an external service.
	ClaimFactory *claim.Factory
	// TrafficMetrics aggregates rate and malformed-packet counters for this server.
	TrafficMetrics *session.TrafficMetrics

	Listeners      []Listener
	StatusProvider minecraft.ServerStatusProvider

	StatusProviderFunc StatusProviderFunc
	ListenerFuncs      []ListenerFunc
	DialerFunc         DialerFunc

	// sessions holds every live *session.Session on this server, keyed by
	// pointer so late-registering sessions with duplicate XUIDs (e.g. a
	// reconnect racing with an old connection) don't clobber one another.
	sessions sync.Map
	xuidMu   sync.Mutex
	xuids    map[string]struct{}

	Log *slog.Logger
}

// ReserveXUID atomically reserves a non-empty XUID until ReleaseXUID is called.
func (s *Server) ReserveXUID(xuid string) bool {
	if xuid == "" {
		return true
	}
	s.xuidMu.Lock()
	defer s.xuidMu.Unlock()
	if s.xuids == nil {
		s.xuids = make(map[string]struct{})
	}
	if _, exists := s.xuids[xuid]; exists {
		return false
	}
	s.xuids[xuid] = struct{}{}
	return true
}

// ReleaseXUID releases an admission reservation.
func (s *Server) ReleaseXUID(xuid string) {
	if xuid == "" {
		return
	}
	s.xuidMu.Lock()
	delete(s.xuids, xuid)
	s.xuidMu.Unlock()
}

// AddSession registers a session with the server for later iteration by the
// AFK evaluator and any other per-server walkers.
func (s *Server) AddSession(sess *session.Session) {
	s.sessions.Store(sess, struct{}{})
}

// RemoveSession deregisters a session from the server.
func (s *Server) RemoveSession(sess *session.Session) {
	s.sessions.Delete(sess)
}

// Sessions returns a snapshot of the server's live sessions.
func (s *Server) Sessions() []*session.Session {
	var out []*session.Session
	s.sessions.Range(func(k, _ any) bool {
		if sess, ok := k.(*session.Session); ok {
			out = append(out, sess)
		}
		return true
	})
	return out
}

// closeListeners closes every listener on the server.
func (s *Server) closeListeners() {
	for _, l := range s.Listeners {
		_ = l.Close()
	}
}

// ListenerFunc creates a Listener.
type ListenerFunc func() (Listener, error)

// StatusProviderFunc creates a status provider.
type StatusProviderFunc func() (minecraft.ServerStatusProvider, error)

// DialerFunc dials a remote server.
type DialerFunc func(identityData login.IdentityData, clientData login.ClientData, ctx context.Context) (session.Conn, error)

const (
	retryInterval = time.Minute
)

// initServer ...
func (gb *GoBDS) initServer(srv *Server) bool {
	for {
		if gb.ctx.Err() != nil {
			return false
		}

		if srv.StatusProvider == nil {
			prov, err := srv.StatusProviderFunc()
			if err != nil {
				srv.Log.Error("failed to create provider", "err", err)
				gb.waitOrAbort(retryInterval)
				continue
			}
			srv.StatusProvider = prov
		}

		if len(srv.Listeners) == 0 {
			if len(srv.ListenerFuncs) == 0 {
				srv.Log.Error("failed to create listener", "err", "no listener funcs")
				gb.waitOrAbort(retryInterval)
				continue
			}
			ls := make([]Listener, 0, len(srv.ListenerFuncs))
			failed := false
			for _, f := range srv.ListenerFuncs {
				l, err := f()
				if err != nil {
					for _, existing := range ls {
						_ = existing.Close()
					}
					srv.Log.Error("failed to create listener", "err", err)
					gb.waitOrAbort(retryInterval)
					failed = true
					break
				}
				ls = append(ls, l)
			}
			if failed {
				continue
			}
			srv.Listeners = ls
		}

		if len(srv.Listeners) > 0 && srv.StatusProvider != nil {
			return true
		}
	}
}

// waitOrAbort ...
func (gb *GoBDS) waitOrAbort(dur time.Duration) {
	select {
	case <-gb.ctx.Done():
	case <-time.After(dur):
	}
}

// claimFetching ...
func (gb *GoBDS) claimFetching(srv *Server) {
	fetch := func() {
		if err := srv.ClaimFactory.Fetch(); err != nil {
			srv.Log.Error("failed to fetch claims", "err", err)
		}
	}
	fetch()
	refresh := time.NewTicker(srv.ClaimFactory.PollInterval())
	defer refresh.Stop()
	const metricPeriod = time.Minute
	metrics := time.NewTicker(metricPeriod)
	defer metrics.Stop()
	for {
		select {
		case <-gb.ctx.Done():
			return
		case <-refresh.C:
			fetch()
		case <-metrics.C:
			srv.ClaimFactory.Metrics().WriteDelta(os.Stdout, metricPeriod, snapshotOf(srv.ClaimFactory))
			srv.TrafficMetrics.WriteDelta(os.Stdout, srv.Name, "", metricPeriod)
			for _, sess := range srv.Sessions() {
				sess.WriteTrafficMetrics(os.Stdout, srv.Name, metricPeriod)
			}
		}
	}
}

func snapshotOf(factory *claim.Factory) *claim.Snapshot {
	snapshot, _ := factory.Snapshot(time.Now())
	return snapshot
}
