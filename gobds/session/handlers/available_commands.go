package handlers

import (
	"slices"
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// AvailableCommands ...
type AvailableCommands struct{}

// disabledCommands ...
var disabledCommands = []string{"me", "tell", "w", "msg"}

var (
	commandsCache []protocol.Command
	commandsMu    sync.RWMutex
)

// Handle ...
func (AvailableCommands) Handle(_ interceptor.Client, pk packet.Packet, _ *session.Context) {
	pkt := pk.(*packet.AvailableCommands)

	filtered := make([]protocol.Command, 0, len(pkt.Commands))
	for _, cmd := range pkt.Commands {
		if !slices.Contains(disabledCommands, cmd.Name) {
			filtered = append(filtered, cmd)
		}
	}
	pkt.Commands = filtered

	if len(commandsCache) == 0 { // ...
		commandsMu.Lock()
		commandsCache = filtered
		commandsMu.Unlock()
	}
}
