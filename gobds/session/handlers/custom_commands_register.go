package handlers

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/cmd"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

type IMinecraftRawText struct {
	Text string `json:"text"`
}

type IMinecraftTextMessage struct {
	RawText []IMinecraftRawText `json:"rawtext"`
}

type CustomCommandRegisterHandler struct{}

// Global variable to store the command file path - set during initialization
var globalCommandPath string

// SetCommandPath sets the global command path - called from gobds.go during initialization
func SetCommandPath(path string) {
	globalCommandPath = path
}

func (CustomCommandRegisterHandler) Handle(_ interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.Text)

	if pkt.TextType != packet.TextTypeObject {
		return
	}

	var messageData IMinecraftTextMessage
	if err := json.Unmarshal([]byte(pkt.Message), &messageData); err != nil {
		log.Println("Failed to parse message JSON", err)
		return
	}
	message := messageData.RawText[0].Text
	if !strings.HasPrefix(message, "[PROXY_SYSTEM][COMMANDS]=") {
		return
	}
	ctx.Cancel() // Ensure client doesn't see the message.
	commandsRaw := strings.TrimPrefix(message, "[PROXY_SYSTEM][COMMANDS]=")

	// The BDS server has sent a custom command register packet
	// We need to rewrite the commands file (config.Resources.CommandPath)
	// Then re-setup commands

	// Parse the JSON commands
	var commands map[string]cmd.EngineResponseCommand
	if err := json.Unmarshal([]byte(commandsRaw), &commands); err != nil {
		log.Println("Failed to parse commands JSON", err)
		return
	}

	// Write commands to file for persistence
	if globalCommandPath != "" {
		// Ensure directory exists
		dir := filepath.Dir(globalCommandPath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			log.Println("Failed to create command directory", err)
			return
		}

		// Write commands to file
		commandsJSON, err := json.MarshalIndent(commands, "", "  ")
		if err != nil {
			log.Println("Failed to marshal commands to JSON", err)
			return
		}

		if err := os.WriteFile(globalCommandPath, commandsJSON, os.ModePerm); err != nil {
			log.Println("Failed to write commands file", err)
			return
		}
	}

	// Reload commands immediately
	cmd.LoadFrom(commands)
	log.Printf("Reloaded %d commands from server", len(commands))
}
