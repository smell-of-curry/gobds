package handlers

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// CommandRequest ...
type CommandRequest struct{}

// Handle ...
func (CommandRequest) Handle(c interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.CommandRequest)

	cmd := strings.ToLower(strings.Split(pkt.CommandLine[1:], " ")[0])
	if slices.Contains(disabledCommands, cmd) {
		ctx.Cancel()
		return
	}
	if strings.ReplaceAll(cmd, " ", "") == "" {
		return
	}

	commandsMu.RLock()
	for _, vanillaCommand := range vanillaCommandsCache {
		if vanillaCommand.Name == cmd {
			commandsMu.RUnlock()
			return
		}
	}
	commandsMu.RUnlock()

	c.WriteToServer(&packet.Text{
		TextType:   packet.TextTypeChat,
		SourceName: c.IdentityData().DisplayName,
		Message:    fmt.Sprintf("-%s", strings.TrimPrefix(pkt.CommandLine, "/")),
		XUID:       c.IdentityData().XUID,
	})
	ctx.Cancel()
}
