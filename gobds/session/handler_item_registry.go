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
	items := make(map[int16]protocol.ItemEntry)
	for _, item := range pkt.Items {
		items[item.RuntimeID] = item
	}
	h.mu.Lock()
	h.items = items
	h.mu.Unlock()
	return nil
}

// Item returns the item registry entry for a runtime ID.
func (h *ItemRegistryHandler) Item(runtimeID int16) (protocol.ItemEntry, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	entry, ok := h.items[runtimeID]
	return entry, ok
}
