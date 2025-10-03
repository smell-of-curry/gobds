package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
)

// RemoveActorHandler ...
type RemoveActorHandler struct{}

// Handle ...
func (*RemoveActorHandler) Handle(_ *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.RemoveActor)
	infra.EntityFactory.RemoveFromRuntimeID(uint64(pkt.EntityUniqueID))
	return nil
}
