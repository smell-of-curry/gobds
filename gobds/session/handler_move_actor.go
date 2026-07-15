package session

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

// MoveActorHandler keeps tracked entity positions current.
type MoveActorHandler struct{}

// Handle ...
func (*MoveActorHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() != s.server {
		return nil
	}
	switch pkt := pk.(type) {
	case *packet.MoveActorAbsolute:
		s.entityFactory.UpdatePosition(pkt.EntityRuntimeID, pkt.Position, true, true, true)
	case *packet.MoveActorDelta:
		s.entityFactory.UpdatePosition(
			pkt.EntityRuntimeID,
			pkt.Position,
			pkt.Flags&packet.MoveActorDeltaFlagHasX != 0,
			pkt.Flags&packet.MoveActorDeltaFlagHasY != 0,
			pkt.Flags&packet.MoveActorDeltaFlagHasZ != 0,
		)
	}
	return nil
}
