package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// ItemRegistry ...
type ItemRegistry struct {
	items map[int16]protocol.ItemEntry
}

// Handle ...
func (i *ItemRegistry) Handle(_ *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.ItemRegistry)
	items := make(map[int16]protocol.ItemEntry)
	for _, item := range pkt.Items {
		items[item.RuntimeID] = item
	}
	i.items = items
	return nil
}
