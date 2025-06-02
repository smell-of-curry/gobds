package handlers

import (
	"strings"

	"github.com/go-jose/go-jose/v4/json"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
	"github.com/smell-of-curry/gobds/gobds/util/command"
)

// Text ...
type Text struct{}

// Handle ...
func (Text) Handle(c interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.Text)
	if pkt.TextType != packet.TextTypeObject {
		return
	}

	var messageData command.MinecraftTextMessage
	if err := json.Unmarshal([]byte(pkt.Message), &messageData); err != nil {
		ctx.Cancel()
		return
	}

	message := messageData.RawText[0].Text
	if !strings.HasPrefix(message, "[PROXY_SYSTEM][COMMANDS]=") {
		return
	}

	commandsRaw := strings.TrimPrefix(message, "[PROXY_SYSTEM][COMMANDS]=")
	var commands map[string]command.EngineResponseCommand
	if err := json.Unmarshal([]byte(commandsRaw), &commands); err != nil {
		ctx.Cancel()
		return
	}

	availableCommandsMu.RLock()
	available := availableCommandsCache
	availableCommandsMu.RUnlock()

	formattedPacket := command.FormatAvailableCommands(commands)
	formattedPacket = command.MergeAvailableCommands(formattedPacket, *available)
	c.WriteToClient(&formattedPacket)
	ctx.Cancel()
}
