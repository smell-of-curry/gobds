package handlers

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// AddPainting ...
type AddPainting struct{}

// Handle ...
func (AddPainting) Handle(_ interceptor.Client, pk packet.Packet, _ *session.Context) {
	pkt := pk.(*packet.AddPainting)
	infra.EntityFactory.Add(entity.NewEntity(pkt.EntityRuntimeID, "minecraft:painting"))
}
