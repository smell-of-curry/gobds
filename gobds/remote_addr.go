package gobds

import (
	"errors"
	"net"
	"net/netip"

	"github.com/df-mc/go-nethernet"
)

// remoteAddrPort returns the player's IP:port for both transports. RakNet
// hands back a *net.UDPAddr; NetherNet hands back a *nethernet.Addr whose
// String() is "<networkID> (<connID>) (<candidate>)" and must be resolved via
// the selected ICE candidate (a prflx/srflx candidate carries the public NAT
// address, which is what the VPN/ban checks need).
func remoteAddrPort(addr net.Addr) (netip.AddrPort, error) {
	if nn, ok := addr.(*nethernet.Addr); ok {
		c := nn.SelectedCandidate
		if c == nil {
			for i := range nn.Candidates {
				if nn.Candidates[i].Address != "" {
					c = &nn.Candidates[i]
					break
				}
			}
		}
		if c == nil {
			return netip.AddrPort{}, errors.New("nethernet: no ICE candidate for remote address")
		}
		ip, err := netip.ParseAddr(c.Address)
		if err != nil {
			return netip.AddrPort{}, err
		}
		return netip.AddrPortFrom(ip.Unmap(), c.Port), nil
	}
	return netip.ParseAddrPort(addr.String())
}
