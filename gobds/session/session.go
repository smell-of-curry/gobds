package session

import (
	"fmt"
	"log/slog"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/util"
)

// Session ...
type Session struct {
	client *minecraft.Conn
	server *minecraft.Conn

	data *Data
	log  *slog.Logger
}

// NewSession ...
func NewSession(client, server *minecraft.Conn, log *slog.Logger) *Session {
	return &Session{
		client: client,
		server: server,

		data: NewData(client),
		log:  log,
	}
}

// pingIdentifier ...
const pingIdentifier = "&_playerPing:"

// SendPingIndicator ...
func (s *Session) SendPingIndicator() {
	ping := s.Ping()

	var colour string
	switch {
	case ping < 20:
		colour = "§a"
	case ping < 50:
		colour = "§e"
	case ping < 100:
		colour = "§6"
	case ping < 200:
		colour = "§c"
	default:
		colour = "§4"
	}

	s.WriteToClient(&packet.SetTitle{
		ActionType: packet.TitleActionSetTitle,
		Text:       fmt.Sprintf("%sCurrent Ping: %s%d", pingIdentifier, colour, ping),
	})
}

// Data ...
func (s *Session) Data() any {
	return s.data
}

// Ping ...
func (s *Session) Ping() int64 {
	return s.client.Latency().Milliseconds()
}

// WriteToClient ...
func (s *Session) WriteToClient(pk packet.Packet) {
	err := s.client.WritePacket(pk)
	if err != nil {
		s.log.Error("error writing to client", "error", err)
	}
}

// WriteToServer ...
func (s *Session) WriteToServer(pk packet.Packet) {
	err := s.server.WritePacket(pk)
	if err != nil {
		s.log.Error("error writing to server", "error", err)
	}
}

// GameData ...
func (s *Session) GameData() minecraft.GameData {
	return s.client.GameData()
}

// ClientData ...
func (s *Session) ClientData() login.ClientData {
	return s.client.ClientData()
}

// Locale ...
func (s *Session) Locale() string {
	return s.ClientData().LanguageCode
}

// IdentityData ...
func (s *Session) IdentityData() login.IdentityData {
	return s.server.IdentityData()
}

// Client ...
func (s *Session) Client() *minecraft.Conn {
	return s.client
}

// Server ...
func (s *Session) Server() *minecraft.Conn {
	return s.server
}

// ForwardXUID ...
func (s *Session) ForwardXUID(encryptionKey string) {
	name, xuid := s.IdentityData().DisplayName, s.IdentityData().XUID
	message := "XUID=" + xuid + " | NAME=" + name

	encryptedMessage, err := util.EncryptMessage(message, encryptionKey)
	if err != nil {
		s.log.Error("error encrypting message", "error", err)
		return
	}
	s.WriteToServer(&packet.Text{
		TextType:         packet.TextTypeChat,
		NeedsTranslation: false,
		SourceName:       s.ClientData().ThirdPartyName,
		Message:          "[PROXY_XUID] " + encryptedMessage,
		XUID:             xuid,
	})
}
