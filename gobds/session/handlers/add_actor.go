package handlers

import (
	"fmt"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/util/translator"
)

// AddActor ...
type AddActor struct{}

// Handle ...
func (AddActor) Handle(c interceptor.Client, pk packet.Packet, _ *session.Context) {
	pkt := pk.(*packet.AddActor)

	entityType := pkt.EntityType
	infra.EntityFactory.Add(entity.NewEntity(pkt.EntityRuntimeID, pkt.EntityType))

	if !strings.HasPrefix(entityType, "pokemon:") {
		return
	}

	name, ok := pkt.EntityMetadata[protocol.EntityDataKeyName]
	if !ok {
		return
	}

	pkt.EntityMetadata[protocol.EntityDataKeyName] = translateName(name.(string), entityType, c)
}

// nickIdentifier ...
const nickIdentifier = "§l§n§r"

// translateName ...
func translateName(name string, entityType string, c interceptor.Client) string {
	if strings.HasPrefix(name, nickIdentifier) {
		return name
	}

	translatedName := entityTranslatedName(entityType, c)

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
func entityTranslatedName(entityType string, c interceptor.Client) string {
	t, ok := translator.TranslationFor(c.Locale())
	if !ok {
		t, _ = translator.TranslationFor("en_US") // Default to american english.
	}
	name, ok := t["item.spawn_egg.entity."+entityType+".name"]
	if !ok {
		return fmt.Sprintf("name not found for %s", entityType)
	}
	return name
}
