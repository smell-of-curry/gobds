package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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
	switch {
	case strings.HasPrefix(message, commandsLegacyPrefix):
		ctx.Cancel()
		return applyCommands(s, strings.TrimPrefix(message, commandsLegacyPrefix))
	case strings.HasPrefix(message, commandsChunkPrefix):
		ctx.Cancel()
		i, n, part, ok := parseCommandChunk(message)
		if !ok {
			s.commandChunks.reset()
			s.log.Error("failed to parse command chunk header")
			return nil
		}
		full, done, err := s.commandChunks.add(i, n, part)
		if err != nil {
			s.log.Error("failed to reassemble command chunks", "error", err, "i", i, "n", n)
			return nil
		}
		if !done {
			return nil
		}
		return applyCommands(s, full)
	default:
		return nil
	}
}

const (
	commandsLegacyPrefix = "[PROXY_SYSTEM][COMMANDS]="
	commandsChunkPrefix  = "[PROXY_SYSTEM][COMMANDS:"

	commandChunkMaxCount = 64
	commandChunkMaxBytes = 4 << 20
)

func parseCommandChunk(message string) (i, n int, part string, ok bool) {
	if !strings.HasPrefix(message, commandsChunkPrefix) {
		return 0, 0, "", false
	}
	rest := message[len(commandsChunkPrefix):]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return 0, 0, "", false
	}
	closeEq := strings.Index(rest[slash+1:], "]=")
	if closeEq < 0 {
		return 0, 0, "", false
	}
	nEnd := slash + 1 + closeEq
	var err error
	i, err = strconv.Atoi(rest[:slash])
	if err != nil {
		return 0, 0, "", false
	}
	n, err = strconv.Atoi(rest[slash+1:nEnd])
	if err != nil {
		return 0, 0, "", false
	}
	return i, n, rest[nEnd+2:], true
}

func applyCommands(s *Session, raw string) error {
	var commands map[string]cmd.EngineResponseCommand
	if err := json.Unmarshal([]byte(raw), &commands); err != nil {
		s.log.Error("failed to parse commands", "error", err)
		return err
	}

	if globalCommandPath != "" {
		dir := filepath.Dir(globalCommandPath)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			s.log.Error("failed to create command directory", "error", err)
			return err
		}

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

	cmd.LoadFrom(commands)
	s.log.Info("reloaded commands from server", "count", len(commands))
	return nil
}

// commandChunks reassembles ordered [PROXY_SYSTEM][COMMANDS:i/n] parts.
type commandChunks struct {
	n     int
	last  int
	parts []string
	bytes int
}

func (c *commandChunks) reset() {
	*c = commandChunks{}
}

func (c *commandChunks) add(i, n int, part string) (full string, done bool, err error) {
	if i < 1 || n < 1 || i > n {
		c.reset()
		return "", false, fmt.Errorf("invalid chunk index %d/%d", i, n)
	}
	if n > commandChunkMaxCount {
		c.reset()
		return "", false, fmt.Errorf("too many chunks: %d", n)
	}
	if i == 1 {
		c.reset()
		c.n = n
	}
	if c.n == 0 || i != c.last+1 || n != c.n {
		c.reset()
		return "", false, fmt.Errorf("unexpected chunk %d/%d", i, n)
	}
	if c.bytes+len(part) > commandChunkMaxBytes {
		c.reset()
		return "", false, fmt.Errorf("command payload exceeds %d bytes", commandChunkMaxBytes)
	}
	c.parts = append(c.parts, part)
	c.bytes += len(part)
	c.last = i
	if i != n {
		return "", false, nil
	}
	full = strings.Join(c.parts, "")
	c.reset()
	return full, true, nil
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
