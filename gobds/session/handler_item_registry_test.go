package session

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

// The ItemRegistry packet is consumed by gophertunnel during DoSpawn, so the
// registry must be seedable from GameData().Items — an unseeded registry made
// the claim stick/shovel exemption dead code (players couldn't check claim
// owners with a stick).
func TestItemRegistrySeedFromGameData(t *testing.T) {
	h := &ItemRegistryHandler{}
	if _, ok := h.Item(328); ok {
		t.Fatal("unseeded registry must miss")
	}
	h.SetItems([]protocol.ItemEntry{
		{Name: "minecraft:stick", RuntimeID: 328},
		{Name: "minecraft:golden_shovel", RuntimeID: 329},
	})
	entry, ok := h.Item(328)
	if !ok || entry.Name != "minecraft:stick" {
		t.Fatalf("seeded registry lookup failed: entry=%v ok=%v", entry, ok)
	}
	if _, ok := h.Item(999); ok {
		t.Fatal("unknown runtime ID must miss")
	}
}
