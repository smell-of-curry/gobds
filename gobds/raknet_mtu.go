package gobds

import (
	"context"
	"log/slog"
	"net"

	"github.com/sandertv/go-raknet"
	"github.com/sandertv/gophertunnel/minecraft"
)

// ponytail: send-path PMTUD start. Handshake still negotiates up to 1492.
// 1200 is HashimTheArab's T-Mobile size. Hub may override per-IP via hint.
const initialSendMTU uint16 = 1200

// pmtudRakNet is gophertunnel's RakNet Network with send-path PMTUD.
// Dialer is left default — gobds↔BDS is same-DC; BDS already caps at 1400.
type pmtudRakNet struct {
	log         *slog.Logger
	initialSend uint16
	hint        func(net.Addr) uint16
}

func (r pmtudRakNet) DialContext(ctx context.Context, address string) (net.Conn, error) {
	return raknet.Dialer{ErrorLog: r.log}.DialContext(ctx, address)
}

func (r pmtudRakNet) PingContext(ctx context.Context, address string) ([]byte, error) {
	return raknet.Dialer{ErrorLog: r.log}.PingContext(ctx, address)
}

func (r pmtudRakNet) Listen(address string) (minecraft.NetworkListener, error) {
	return raknet.ListenConfig{
		ErrorLog:           r.log,
		InitialSendMTU:     r.initialSend,
		InitialSendMTUFunc: r.hint,
	}.Listen(address)
}

func ipString(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	if a, ok := addr.(*net.UDPAddr); ok {
		return a.IP.String()
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
