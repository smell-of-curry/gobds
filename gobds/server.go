package gobds

import (
	"context"
	"log/slog"

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
