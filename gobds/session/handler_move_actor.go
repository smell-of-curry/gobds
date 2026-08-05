package session

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

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
		x, hasX := pkt.PositionX.Value()
		y, hasY := pkt.PositionY.Value()
		z, hasZ := pkt.PositionZ.Value()
		s.entityFactory.UpdatePosition(
			pkt.EntityRuntimeID,
			mgl32.Vec3{x, y, z},
			hasX,
			hasY,
			hasZ,
		)
	}
	return nil
}
