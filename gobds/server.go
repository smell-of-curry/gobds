package gobds

import (
	"context"
	"log/slog"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/entity"
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

	EntityFactory *entity.Factory
	ClaimFactory  *claim.Factory

	Listener       Listener
	StatusProvider minecraft.ServerStatusProvider

	StatusProviderFunc StatusProviderFunc
	ListenerFunc       ListenerFunc
	DialerFunc         DialerFunc

	Log *slog.Logger
}

// ListenerFunc creates a Listener.
type ListenerFunc func() (Listener, error)

// StatusProviderFunc creates a status provider.
type StatusProviderFunc func() (minecraft.ServerStatusProvider, error)

// DialerFunc dials a remote server.
type DialerFunc func(identityData login.IdentityData, clientData login.ClientData, ctx context.Context) (session.Conn, error)

const (
	maxRetries    = 30
	retryInterval = time.Minute
)

// initServer ...
func (gb *GoBDS) initServer(srv *Server) bool {
	for retry := 0; retry < maxRetries; retry++ {
		if gb.ctx.Err() != nil {
			return false
		}

		if srv.StatusProvider == nil {
			prov, err := srv.StatusProviderFunc()
			if err != nil {
				srv.Log.Error("failed to create provider", "err", err)
				if retry < maxRetries-1 {
					gb.waitOrAbort(retryInterval)
					continue
				}
				return false
			}
			srv.StatusProvider = prov
		}

		if srv.Listener == nil {
			l, err := srv.ListenerFunc()
			if err != nil {
				srv.Log.Error("failed to create listener", "err", err)
				if retry < maxRetries-1 {
					gb.waitOrAbort(retryInterval)
					continue
				}
				return false
			}
			srv.Listener = l
		}

		if srv.Listener != nil && srv.StatusProvider != nil {
			return true
		}
	}
	srv.Log.Error("failed to init server", "addr", srv.LocalAddress)
	return false
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
