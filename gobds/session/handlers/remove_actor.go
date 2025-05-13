package handlers

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// RemoveActor ...
type RemoveActor struct{}

// Handle ...
func (RemoveActor) Handle(_ interceptor.Client, pk packet.Packet, _ *session.Context) {
	pkt := pk.(*packet.RemoveActor)
	infra.EntityFactory.RemoveFromRuntimeID(uint64(pkt.EntityUniqueID))
}
