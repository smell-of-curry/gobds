package gobds

import (
	"context"
	"log/slog"
	"net"

	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft"
)

// ponytail: gold is a different hop than the hub. WARP/hotspot often uses
// another egress IP, and a hub-proven 1400/1492 send MTU dies on this path.
// Cap handshake at 1200 (Hashim / T-Mobile / WARP). Do not apply hub hints.
// Ceiling: ~25% more datagrams vs 1492. Climb later if a per-dest probe exists.
const warpSafeMTU uint16 = 1200

// cappedRakNet is gophertunnel's RakNet Network with a listener MTU cap.
// Dialer stays default — gobds↔BDS is same-DC; BDS already caps at 1400.
type cappedRakNet struct {
	log *slog.Logger
	mtu uint16
}

func (r cappedRakNet) DialContext(ctx context.Context, address string) (net.Conn, error) {
	return raknet.Dialer{ErrorLog: r.log}.DialContext(ctx, address)
}

func (r cappedRakNet) PingContext(ctx context.Context, address string) ([]byte, error) {
	return raknet.Dialer{ErrorLog: r.log}.PingContext(ctx, address)
}

func (r cappedRakNet) Listen(address string) (minecraft.NetworkListener, error) {
	// ponytail: DisableCookies — OVH/WARP can rewrite the UDP 5-tuple
	// between OCR1 and OCR2; cookie mismatch = NetherNet, no login.
	return raknet.ListenConfig{ErrorLog: r.log, MaxMTU: r.mtu, DisableCookies: true}.Listen(address)
}
