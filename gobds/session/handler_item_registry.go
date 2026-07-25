package session

import (
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ItemRegistryHandler ...
type ItemRegistryHandler struct {
	items map[int16]protocol.ItemEntry
	mu    sync.RWMutex
}

// Handle ...
func (h *ItemRegistryHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() != s.server {
		return nil
	}
	pkt := pk.(*packet.ItemRegistry)
	h.SetItems(pkt.Items)
	return nil
}

// SetItems replaces the registry contents. Sessions must seed this from the
// negotiated GameData: gophertunnel consumes the login-sequence ItemRegistry
// packet inside DoSpawn (exposing it only as GameData().Items), so it never
// reaches handlePacket — without seeding, held-item lookups always miss and
// the claim stick/shovel exemptions and item-drop denials silently no-op.
func (h *ItemRegistryHandler) SetItems(entries []protocol.ItemEntry) {
	items := make(map[int16]protocol.ItemEntry, len(entries))
	for _, item := range entries {
		items[item.RuntimeID] = item
	}
	h.mu.Lock()
	h.items = items
	h.mu.Unlock()
}

// Item returns the item registry entry for a runtime ID.
func (h *ItemRegistryHandler) Item(runtimeID int16) (protocol.ItemEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	entry, ok := h.items[runtimeID]
	return entry, ok
}
