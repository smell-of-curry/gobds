package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/entity"
)

// AddPaintingHandler ...
type AddPaintingHandler struct{}

// Handle ...
func (*AddPaintingHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() != s.server {
		return nil
	}
	pkt := pk.(*packet.AddPainting)
	s.entityFactory.Add(entity.NewEntity(
		pkt.EntityUniqueID,
		pkt.EntityRuntimeID,
		"minecraft:painting",
		pkt.Position,
	))
	return nil
}
