package gobds

import (
	"context"
	"log/slog"
	"net"

	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft"
)

// loggedRakNet is gophertunnel's RakNet Network with ErrorLog wired through.
type loggedRakNet struct {
	log *slog.Logger
}

func (r loggedRakNet) DialContext(ctx context.Context, address string) (net.Conn, error) {
	return raknet.Dialer{ErrorLog: r.log}.DialContext(ctx, address)
}

func (r loggedRakNet) PingContext(ctx context.Context, address string) ([]byte, error) {
	return raknet.Dialer{ErrorLog: r.log}.PingContext(ctx, address)
}

func (r loggedRakNet) Listen(address string) (minecraft.NetworkListener, error) {
	return raknet.ListenConfig{ErrorLog: r.log}.Listen(address)
}
