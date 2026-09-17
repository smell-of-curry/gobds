package gobds

import "github.com/sandertv/gophertunnel/minecraft"

// proxyStatusProvider is a minecraft.ServerStatusProvider that reports the
// proxy's own live player count and a statically configured capacity/name
// instead of mirroring the backend BDS unconnected pong.
//
// Mirroring the backend (minecraft.ForeignStatusProvider) requires the backend
// to answer unconnected pings with a parseable server-list payload. Recent BDS
// builds running with enable-lan-visibility=false reply with a header-only pong
// that carries no payload, which the RakNet client rejects ("unexpected EOF");
// the proxy then served an empty 0/1 status forever. Reporting locally removes
// that dependency entirely, is more accurate (the proxy is the authoritative
// owner of the live session set), and avoids parsing untrusted backend bytes on
// a one-second timer.
type proxyStatusProvider struct {
	// srv is the server whose live sessions back the player count when the
	// caller does not supply one (a negative playerCount).
	srv *Server
	// name is the MOTD/server name shown in the server list.
	name string
	// maxPlayers is the capacity advertised in the server list.
	maxPlayers int
}

// newProxyStatusProvider creates a proxyStatusProvider for the server passed.
//
// srv is the server whose sessions are counted, name is the MOTD shown in the
// list, and maxPlayers is the advertised capacity.
func newProxyStatusProvider(srv *Server, name string, maxPlayers int) *proxyStatusProvider {
	return &proxyStatusProvider{srv: srv, name: name, maxPlayers: maxPlayers}
}

// ServerStatus reports the current status of the proxy.
//
// Both parameters (the calling listener's own player count and max players)
// are intentionally ignored: the server's tracked session count and the
// configured capacity are authoritative across all listeners.
// It returns the status advertised in the server list.
func (p *proxyStatusProvider) ServerStatus(_, _ int) minecraft.ServerStatus {
	// Always report the server's own session count: with RakNet and NetherNet
	// listeners side by side, each listener only knows its own connections, so
	// the RakNet pong (what the hub / Discord poll) showed 0 while everyone was
	// on NetherNet.
	count := len(p.srv.Sessions())
	return minecraft.ServerStatus{
		ServerName:  p.name,
		PlayerCount: count,
		MaxPlayers:  p.maxPlayers,
	}
}
