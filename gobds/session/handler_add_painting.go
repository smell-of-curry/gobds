package session

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/entity"
)

// AddPaintingHandler ...
type AddPaintingHandler struct{}

// Handle ...
func (*AddPaintingHandler) Handle(s *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.AddPainting)
	s.entityFactory.Add(entity.NewEntity(pkt.EntityRuntimeID, "minecraft:painting", mgl32.Vec3{
		pkt.Position.X(), pkt.Position.Y(), pkt.Position.Z(),
	}))
	return nil
}
