package session

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// SetPlayerGameTypeHandler ...
type SetPlayerGameTypeHandler struct{}

// Handle ...
func (*SetPlayerGameTypeHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.SetPlayerGameType)
	if ctx.Val() != s.server {
		return nil
	}
	s.Data().SetGameMode(pkt.GameType)
	return nil
}
