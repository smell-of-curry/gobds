package session

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// UpdatePlayerGameTypeHandler ...
type UpdatePlayerGameTypeHandler struct{}

// Handle ...
func (*UpdatePlayerGameTypeHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.UpdatePlayerGameType)
	if ctx.Val() != s.server {
		return nil
	}
	s.Data().SetGameMode(pkt.GameType)
	return nil
}
