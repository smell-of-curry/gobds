package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ItemRegistryHandler ...
type ItemRegistryHandler struct {
	items map[int16]protocol.ItemEntry
}

// Handle ...
func (h *ItemRegistryHandler) Handle(_ *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.ItemRegistry)
	items := make(map[int16]protocol.ItemEntry)
	for _, item := range pkt.Items {
		items[item.RuntimeID] = item
	}
	h.items = items
	return nil
}
