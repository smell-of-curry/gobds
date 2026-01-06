package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// RemoveActorHandler ...
type RemoveActorHandler struct{}

// Handle ...
func (*RemoveActorHandler) Handle(s *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.RemoveActor)
	s.entityFactory.RemoveFromRuntimeID(uint64(pkt.EntityUniqueID))
	return nil
}
