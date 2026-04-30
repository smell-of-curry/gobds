package gobds

import (
	"context"
	"log/slog"
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
	ClaimService  struct {
		Enabled bool
		URL     string
		Key     string
	}
}

// Server represents a single server instance with its own Listener, DialerFunc, and minecraft.ServerStatusProvider.
type Server struct {
	Name          string
	LocalAddress  string
	RemoteAddress string

	// ClaimFactory is shared across all sessions on this server because claims are world-state
	// fetched periodically from an external service.
	ClaimFactory *claim.Factory

	Listener       Listener
	StatusProvider minecraft.ServerStatusProvider

	StatusProviderFunc StatusProviderFunc
	ListenerFunc       ListenerFunc
	DialerFunc         DialerFunc

	// sessions holds every live *session.Session on this server, keyed by
	// pointer so late-registering sessions with duplicate XUIDs (e.g. a
	// reconnect racing with an old connection) don't clobber one another.
	sessions sync.Map

	Log *slog.Logger
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

		if srv.Listener == nil {
			l, err := srv.ListenerFunc()
			if err != nil {
				srv.Log.Error("failed to create listener", "err", err)
				gb.waitOrAbort(retryInterval)
				continue
			}
			srv.Listener = l
		}

		if srv.Listener != nil && srv.StatusProvider != nil {
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
	t := time.NewTicker(5 * time.Minute)
	defer t.Stop()
	for {
		select {
		case <-gb.ctx.Done():
			return
		case <-t.C:
			fetch()
		}
	}
}
