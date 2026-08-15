package gobds

import (
	"context"
	"log/slog"
	"net"

	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft"
)

// ponytail: handshake still negotiates up to 1492; only the send path starts
// small and climbs 1280 → 1400 → 1492 as probes are ACKed. Keeps low-MTU paths
// (WARP, T-Mobile, hotspots) usable without pinning every client to 1200.
// Ceiling: a few probe datagrams per connection.
const initialSendMTU uint16 = 1200

// cappedRakNet is gophertunnel's RakNet Network with send-path PMTUD.
// Dialer stays default — gobds↔BDS is same-DC; BDS already caps at 1400.
type cappedRakNet struct {
	log         *slog.Logger
	initialSend uint16
}

func (r cappedRakNet) DialContext(ctx context.Context, address string) (net.Conn, error) {
	return raknet.Dialer{ErrorLog: r.log}.DialContext(ctx, address)
}

func (r cappedRakNet) PingContext(ctx context.Context, address string) ([]byte, error) {
	return raknet.Dialer{ErrorLog: r.log}.PingContext(ctx, address)
}

func (r cappedRakNet) Listen(address string) (minecraft.NetworkListener, error) {
	return raknet.ListenConfig{ErrorLog: r.log, InitialSendMTU: r.initialSend}.Listen(address)
}
