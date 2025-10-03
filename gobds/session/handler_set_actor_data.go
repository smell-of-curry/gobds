package session

import (
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
)

// SetActorDataHandler ...
type SetActorDataHandler struct{}

// Handle ...
func (*SetActorDataHandler) Handle(s *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.SetActorData)

	ent, ok := infra.EntityFactory.ByRuntimeID(pkt.EntityRuntimeID)
	if !ok {
		return nil
	}

	entityType := ent.ActorType()
	if !strings.HasPrefix(entityType, "pokemon:") {
		return nil
	}

	name, ok := pkt.EntityMetadata[protocol.EntityDataKeyName]
	if !ok {
		return nil
	}

	pkt.EntityMetadata[protocol.EntityDataKeyName] = translateName(name.(string), entityType, s)
	return nil
}
