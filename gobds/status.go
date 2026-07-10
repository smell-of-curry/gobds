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
// The Listener calls this with its own live connection count as playerCount;
// internal callers (e.g. the AFK evaluator) pass a negative playerCount, in
// which case the authoritative live session count of the server is used. The
// caller's maxPlayers is ignored in favor of the configured capacity.
//
// playerCount is the listener's live connection count, or negative to request
// the server's tracked session count.
// The second parameter (the caller's max players) is intentionally ignored.
// It returns the status advertised in the server list.
func (p *proxyStatusProvider) ServerStatus(playerCount, _ int) minecraft.ServerStatus {
	count := playerCount
	if count < 0 {
		count = len(p.srv.Sessions())
	}
	if count < 0 {
		count = 0
	}
	return minecraft.ServerStatus{
		ServerName:  p.name,
		PlayerCount: count,
		MaxPlayers:  p.maxPlayers,
	}
}
