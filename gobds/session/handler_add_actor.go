package session

import (
	"fmt"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/util/translator"
)

// AddActorHandler ...
type AddActorHandler struct{}

// Handle ...
func (*AddActorHandler) Handle(s *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.AddActor)

	entityType := pkt.EntityType
	infra.EntityFactory.Add(entity.NewEntity(pkt.EntityRuntimeID, entityType, pkt.Position))

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

// nickIdentifier ...
const nickIdentifier = "§l§n§r"

// translateName ...
func translateName(name string, entityType string, s *Session) string {
	if strings.HasPrefix(name, nickIdentifier) {
		return name
	}

	translatedName := entityTranslatedName(entityType, s)

	name = strings.TrimSpace(name)
	lines := strings.Split(name, "\n")

	if len(lines) > 0 {
		if strings.Contains(lines[0], "§eLvl") {
			parts := strings.Split(lines[0], "§eLvl")
			lines[0] = "§l" + translatedName + " §eLvl" + parts[1]
		} else {
			lines[0] = "§l" + translatedName
		}
		name = strings.Join(lines, "\n")
	} else {
		name = "§l" + translatedName
	}
	return name
}

// entityTranslatedName ...
func entityTranslatedName(entityType string, s *Session) string {
	t, ok := translator.TranslationFor(s.Locale())
	if !ok {
		t, _ = translator.TranslationFor("en_US") // Default to american english.
	}
	name, ok := t["item.spawn_egg.entity."+entityType+".name"]
	if !ok {
		return fmt.Sprintf("name not found for %s", entityType)
	}
	return name
}
