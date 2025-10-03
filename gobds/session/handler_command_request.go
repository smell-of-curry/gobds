package session

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// CommandRequestHandler ...
type CommandRequestHandler struct{}

// Handle ...
func (*CommandRequestHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.CommandRequest)

	cmd := strings.ToLower(strings.Split(pkt.CommandLine[1:], " ")[0])
	if slices.Contains(disabledCommands, cmd) {
		ctx.Cancel()
		return nil
	}
	if strings.ReplaceAll(cmd, " ", "") == "" {
		return nil
	}

	handler := s.handlers[packet.IDAvailableCommands].(*AvailableCommandsHandler)
	_, ok := handler.cache.Load(cmd)
	if ok {
		return nil
	}

	s.WriteToServer(&packet.Text{
		TextType:   packet.TextTypeChat,
		SourceName: s.IdentityData().DisplayName,
		Message:    fmt.Sprintf("-%s", strings.TrimPrefix(pkt.CommandLine, "/")),
		XUID:       s.IdentityData().XUID,
	})
	ctx.Cancel()
	return nil
}
