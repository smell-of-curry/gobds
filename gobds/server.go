package gobds

import (
	"context"
	"log/slog"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// Server represents a single server instance with its own Listener, DialerFunc, and minecraft.ServerStatusProvider.
type Server struct {
	Name          string
	LocalAddress  string
	RemoteAddress string

	Listener       Listener
	DialerFunc     DialerFunc
	StatusProvider minecraft.ServerStatusProvider
	Log            *slog.Logger
}

// DialerFunc dials a remote server.
type DialerFunc func(identityData login.IdentityData, clientData login.ClientData, ctx context.Context) (session.Conn, error)
