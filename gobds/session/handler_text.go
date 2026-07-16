package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/cmd"
)

// IMinecraftRawText represents a raw text component in Minecraft messages.
type IMinecraftRawText struct {
	Text string `json:"text"`
}

// IMinecraftTextMessage represents a complete Minecraft text message.
type IMinecraftTextMessage struct {
	RawText []IMinecraftRawText `json:"rawtext"`
}

// TextHandler handles text-based packets from the server.
type TextHandler struct{}

// Global variable to store the command file path - set during initialization
var globalCommandPath string

// SetCommandPath sets the global command path - called from gobds.go during initialization
func SetCommandPath(path string) {
	globalCommandPath = path
}

// Handle processes text packets from the server and executes commands.
func (*TextHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.Text)

	if ctx.Val() == s.client {
		return handleClientText(s, pkt, ctx)
	}

	if pkt.TextType != packet.TextTypeObject {
		return nil
	}

	var messageData IMinecraftTextMessage
	if err := json.Unmarshal([]byte(pkt.Message), &messageData); err != nil {
		s.log.Error("failed to parse message", "error", err)
		return err
	}
	if len(messageData.RawText) == 0 {
		return nil
	}
	message := messageData.RawText[0].Text
	if !strings.HasPrefix(message, "[PROXY_SYSTEM][COMMANDS]=") {
		return nil
	}
	ctx.Cancel() // Ensure client doesn't see the message.
	commandsRaw := strings.TrimPrefix(message, "[PROXY_SYSTEM][COMMANDS]=")

	// The BDS server has sent a custom command register packet
	// We need to rewrite the commands file (config.Resources.CommandPath)
	// Then re-setup commands

	// Parse the JSON commands
	var commands map[string]cmd.EngineResponseCommand
	if err := json.Unmarshal([]byte(commandsRaw), &commands); err != nil {
		s.log.Error("failed to parse commands", "error", err)
		return err
	}

	// Write commands to file for persistence
	if globalCommandPath != "" {
		// Ensure directory exists
		dir := filepath.Dir(globalCommandPath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			s.log.Error("failed to create command directory", "error", err)
			return err
		}

		// Write commands to file
		commandsJSON, err := json.MarshalIndent(commands, "", "  ")
		if err != nil {
			s.log.Error("failed to marshal commands", "error", err)
			return err
		}

		if err = os.WriteFile(globalCommandPath, commandsJSON, os.ModePerm); err != nil {
			s.log.Error("failed to write commands file", "error", err)
			return err
		}
	}

	// Reload commands immediately
	cmd.LoadFrom(commands)
	s.log.Info("reloaded commands from server", "count", len(commands))
	return nil
}

func handleClientText(s *Session, pkt *packet.Text, ctx *Context) error {
	if len(pkt.Message) > s.traffic.config.MaxTextBytes || len(pkt.SourceName) > 256 ||
		len(pkt.Parameters) > 64 {
		s.traffic.malformed(trafficChat)
		return malformedPacketError{reason: "text payload exceeds maximum length"}
	}
	for _, parameter := range pkt.Parameters {
		if len(parameter) > s.traffic.config.MaxTextBytes {
			s.traffic.malformed(trafficChat)
			return malformedPacketError{reason: "text parameter exceeds maximum length"}
		}
	}
	if strings.TrimSpace(pkt.Message) == "" {
		s.traffic.malformed(trafficChat)
		ctx.Cancel()
		return nil
	}
	if !s.traffic.allow(trafficChat) {
		ctx.Cancel()
	}
	return nil
}
