package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/cmd"
)

// IMinecraftRawText ...
type IMinecraftRawText struct {
	Text string `json:"text"`
}

// IMinecraftTextMessage ...
type IMinecraftTextMessage struct {
	RawText []IMinecraftRawText `json:"rawtext"`
}

// TextHandler ...
type TextHandler struct{}

// Global variable to store the command file path - set during initialization
var globalCommandPath string

// SetCommandPath sets the global command path - called from gobds.go during initialization
func SetCommandPath(path string) {
	globalCommandPath = path
}

// Handle ...
func (*TextHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.Text)

	// ensuring that only server packets are processed
	if pkt.TextType != packet.TextTypeObject || ctx.Val() != s.server {
		return nil
	}

	var messageData IMinecraftTextMessage
	if err := json.Unmarshal([]byte(pkt.Message), &messageData); err != nil {
		s.log.Error("failed to parse message", "error", err)
		return err
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
		s.log.Error("failed to parse commands", err)
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
	s.log.Info("reloaded commands from remote server", "count", len(commands))
	return nil
}
