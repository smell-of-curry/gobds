package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// RemoveActorHandler ...
type RemoveActorHandler struct{}

// Handle ...
func (*RemoveActorHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() != s.server {
		return nil
	}
	pkt := pk.(*packet.RemoveActor)
	s.entityFactory.RemoveFromUniqueID(pkt.EntityUniqueID)
	return nil
}
