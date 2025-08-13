package handlers

import (
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// SetActorData ...
type SetActorData struct{}

// Handle ...
func (SetActorData) Handle(c interceptor.Client, pk packet.Packet, _ *session.Context) {
	pkt := pk.(*packet.SetActorData)

	ent, ok := infra.EntityFactory.ByRuntimeID(pkt.EntityRuntimeID)
	if !ok {
		return
	}

	entityType := ent.ActorType()
	if !strings.HasPrefix(entityType, "pokemon:") {
		return
	}

	// name, ok := pkt.EntityMetadata[protocol.EntityDataKeyName]
	// if !ok {
	// 	return
	// }

	// pkt.EntityMetadata[protocol.EntityDataKeyName] = translateName(name.(string), entityType, c)
}
