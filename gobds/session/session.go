package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/util"
	"github.com/smell-of-curry/gobds/gobds/util/area"
)

// Session ...
type Session struct {
	client   Conn
	server   Conn
	handlers map[uint32]packetHandler

	pingIndicator *infra.PingIndicator
	afkTimer      *infra.AFKTimer
	border        *area.Area2D

	close chan struct{}

	data *Data
	log  *slog.Logger
}

// SendPingIndicator ...
func (s *Session) SendPingIndicator() {
	if s.pingIndicator == nil {
		return
	}

	ping := s.Ping()
	var color string
	switch {
	case ping < 20:
		color = "§a"
	case ping < 50:
		color = "§e"
	case ping < 100:
		color = "§6"
	case ping < 200:
		color = "§c"
	default:
		color = "§4"
	}

	s.WriteToClient(&packet.SetTitle{
		ActionType: packet.TitleActionSetTitle,
		Text:       fmt.Sprintf("%sCurrent Ping: %s%d", s.pingIndicator.Identifier, color, ping),
	})
}

// Data ...
func (s *Session) Data() *Data {
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

// ReadPackets reads and processes all packets.
func (s *Session) ReadPackets(ctx context.Context) {
	defer close(s.close)
	s.wait(ctx)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer s.server.Close()
		defer wg.Done()
		for {
			pk, err := s.client.ReadPacket()
			if err != nil {
				return
			}
			send, err := s.handlePacket(pk, s.client)
			if err != nil {
				return
			}
			if send {
				_ = s.server.WritePacket(pk)
			}
		}
	}()

	go func() {
		defer s.client.Close()
		defer wg.Done()
		for {
			pk, err := s.server.ReadPacket()
			if err != nil {
				return
			}
			send, err := s.handlePacket(pk, s.server)
			if err != nil {
				return
			}
			if send {
				_ = s.client.WritePacket(pk)
			}
		}
	}()
	wg.Wait()
}

// wait waits until the proxy closes or the client disconnects.
func (s *Session) wait(ctx context.Context) {
	go func() {
		select {
		case <-s.close:
		case <-ctx.Done():
			s.Disconnect("proxy closed")
		}
	}()
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

// Message ...
func (s *Session) Message(message string) {
	_ = s.client.WritePacket(&packet.Text{
		TextType: packet.TextTypeRaw,
		Message:  message,
	})
}

// Disconnect ...
func (s *Session) Disconnect(message string) {
	_ = s.client.WritePacket(&packet.Disconnect{
		HideDisconnectionScreen: message == "",
		Message:                 message,
	})
	_ = s.client.Close()
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
func (s *Session) Client() session.Conn {
	return s.client
}

// Server ...
func (s *Session) Server() session.Conn {
	return s.server
}

// handlePacket passes packet into corresponding handler.
func (s *Session) handlePacket(p packet.Packet, conn Conn) (bool, error) {
	handler, ok := s.handlers[p.ID()]
	if ok {
		ctx := event.C(conn)
		err := handler.Handle(s, p, ctx)
		if err != nil {
			s.log.Error("error handling packet", "packet", p, "error", err)
			s.Disconnect(err.Error())
		}
		return !ctx.Cancelled(), err
	}
	return true, nil
}

// registerHandlers registers all packet handlers.
func (s *Session) registerHandlers() {
	s.handlers = map[uint32]packetHandler{
		packet.IDAddActor:             &AddActorHandler{},
		packet.IDAddPainting:          &AddPaintingHandler{},
		packet.IDAvailableCommands:    &AvailableCommandsHandler{},
		packet.IDCommandRequest:       &CommandRequestHandler{},
		packet.IDInventoryTransaction: &InventoryTransactionHandler{},
		packet.IDItemRegistry:         &ItemRegistryHandler{},
		packet.IDItemStackRequest:     &ItemStackRequestHandler{},
		packet.IDLevelChunk:           &LevelChunkHandler{},
		packet.IDPlayerAuthInput:      NewPlayerAuthInputHandler(),
		packet.IDRemoveActor:          &RemoveActorHandler{},
		packet.IDSetActorData:         &SetActorDataHandler{},
		packet.IDSubChunk:             &SubChunkHandler{},
		packet.IDText:                 &TextHandler{},
	}
}
