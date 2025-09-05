package session

import (
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
)

// SetActorDataHandler ...
type SetActorDataHandler struct{}

// Handle ...
func (*SetActorDataHandler) Handle(c *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.SetActorData)

	ent, ok := infra.EntityFactory.ByRuntimeID(pkt.EntityRuntimeID)
	if !ok {
		return nil
	}

	entityType := ent.ActorType()
	if !strings.HasPrefix(entityType, "pokemon:") {
		return nil
	}

	// name, ok := pkt.EntityMetadata[protocol.EntityDataKeyName]
	// if !ok {
	// 	return
	// }

	// pkt.EntityMetadata[protocol.EntityDataKeyName] = translateName(name.(string), entityType, c)
	return nil
}
