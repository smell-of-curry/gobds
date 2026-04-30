package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/util"
	"github.com/smell-of-curry/gobds/gobds/util/area"
)

// Session ...
type Session struct {
	client   Conn
	server   Conn
	handlers map[uint32]packetHandler

	entityFactory *entity.Factory
	claimFactory  *claim.Factory

	pingIndicator *infra.PingIndicator
	afkTimer      *infra.AFKTimer
	border        *area.Area2D

	close chan struct{}

	afk afkState

	data *Data
	log  *slog.Logger
}

// afkState tracks per-session AFK bookkeeping used by the AFK evaluator.
//
// The last position/rotation are stored here so the packet handler can be a
// pure movement tracker without holding private state that the evaluator
// goroutine cannot read.
type afkState struct {
	mu sync.RWMutex

	lastMoveTime time.Time
	lastPosition mgl32.Vec3
	lastYaw      float32
	lastPitch    float32

	warnedApproaching bool
	markedAFK         bool
	warnedFinal       bool
}

// AFKTimer returns the session's AFK configuration, or nil if disabled.
func (s *Session) AFKTimer() *infra.AFKTimer {
	return s.afkTimer
}

// TouchMovement updates the last movement bookkeeping for the session. When
// the position or orientation has changed it also resets the AFK warning
// flags so the next idle streak starts from scratch. Returns true if the
// session was considered to have moved.
func (s *Session) TouchMovement(pos mgl32.Vec3, yaw, pitch float32) bool {
	s.afk.mu.Lock()
	defer s.afk.mu.Unlock()

	moved := !s.afk.lastPosition.ApproxEqual(pos) ||
		!mgl32.FloatEqual(s.afk.lastYaw, yaw) ||
		!mgl32.FloatEqual(s.afk.lastPitch, pitch)
	if !moved {
		return false
	}

	s.afk.lastMoveTime = time.Now()
	s.afk.lastPosition = pos
	s.afk.lastYaw = yaw
	s.afk.lastPitch = pitch
	s.afk.warnedApproaching = false
	s.afk.markedAFK = false
	s.afk.warnedFinal = false
	return true
}

// AFKDuration returns how long the session has been idle. Before the first
// movement packet is received the session is considered to have just started
// so the duration is measured from session creation.
func (s *Session) AFKDuration() time.Duration {
	s.afk.mu.RLock()
	defer s.afk.mu.RUnlock()

	if s.afk.lastMoveTime.IsZero() {
		return 0
	}
	return time.Since(s.afk.lastMoveTime)
}

// WarnedApproaching returns whether the approaching-AFK soft warning has
// already been sent for the current idle streak.
func (s *Session) WarnedApproaching() bool {
	s.afk.mu.RLock()
	defer s.afk.mu.RUnlock()
	return s.afk.warnedApproaching
}

// SetWarnedApproaching marks the approaching-AFK soft warning as sent.
func (s *Session) SetWarnedApproaching(v bool) {
	s.afk.mu.Lock()
	s.afk.warnedApproaching = v
	s.afk.mu.Unlock()
}

// MarkedAFK returns whether the "you are now AFK" soft warning has already
// been sent for the current idle streak.
func (s *Session) MarkedAFK() bool {
	s.afk.mu.RLock()
	defer s.afk.mu.RUnlock()
	return s.afk.markedAFK
}

// SetMarkedAFK marks the "you are now AFK" soft warning as sent.
func (s *Session) SetMarkedAFK(v bool) {
	s.afk.mu.Lock()
	s.afk.markedAFK = v
	s.afk.mu.Unlock()
}

// WarnedFinal returns whether the near-capacity final warning has already
// been sent for the current idle streak.
func (s *Session) WarnedFinal() bool {
	s.afk.mu.RLock()
	defer s.afk.mu.RUnlock()
	return s.afk.warnedFinal
}

// SetWarnedFinal marks the near-capacity final warning as sent.
func (s *Session) SetWarnedFinal(v bool) {
	s.afk.mu.Lock()
	s.afk.warnedFinal = v
	s.afk.mu.Unlock()
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
		defer func() { _ = s.server.Close() }()
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
		defer func() { _ = s.client.Close() }()
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
