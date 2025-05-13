package handlers

import (
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// ItemRegistry ...
type ItemRegistry struct{}

var (
	itemsCache []protocol.ItemEntry
	itemsMu    sync.RWMutex
)

// Handle ...
func (ItemRegistry) Handle(_ interceptor.Client, pk packet.Packet, _ *session.Context) {
	pkt := pk.(*packet.ItemRegistry)

	if len(itemsCache) == 0 { // ...
		itemsMu.Lock()
		itemsCache = pkt.Items
		itemsMu.Unlock()
	}
}
