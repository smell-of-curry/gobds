package session

import (
	"fmt"
	"slices"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/cmd"
)

// CommandRequestHandler ...
type CommandRequestHandler struct{}

// Handle ...
func (*CommandRequestHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.CommandRequest)
	if ctx.Val() != s.client {
		return nil
	}
	name, empty, err := commandName(pkt.CommandLine, s.traffic.config.MaxCommandBytes)
	if err != nil {
		s.traffic.malformed(trafficCommand)
		return err
	}
	if empty {
		s.traffic.malformed(trafficCommand)
		ctx.Cancel()
		return nil
	}
	if !s.traffic.allow(trafficCommand) {
		ctx.Cancel()
		return nil
	}
	if name == "" {
		return nil
	}

	if slices.Contains(disabledCommands, name) {
		ctx.Cancel()
		return nil
	}

	handler := s.handlers[packet.IDAvailableCommands].(*AvailableCommandsHandler)
	_, vanilla := handler.cache.Load(name)
	// The pack catalog can arrive after AvailableCommands cached the vanilla name.
	if vanilla && !cmd.Shadows(name) {
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

func commandName(line string, maxBytes int) (name string, empty bool, err error) {
	if len(line) > maxBytes {
		return "", false, malformedPacketError{reason: "command exceeds maximum length"}
	}
	line = strings.TrimSpace(line)
	if line == "" || line == "/" {
		return "", true, nil
	}
	if line[0] != '/' {
		return "", false, nil
	}
	name, _, _ = strings.Cut(strings.ToLower(line[1:]), " ")
	return name, false, nil
}
