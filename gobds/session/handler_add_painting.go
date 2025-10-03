package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
)

// AddPaintingHandler ...
type AddPaintingHandler struct{}

// Handle ...
func (*AddPaintingHandler) Handle(_ *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.AddPainting)
	infra.EntityFactory.Add(entity.NewEntity(pkt.EntityRuntimeID, "minecraft:painting"))
	return nil
}
