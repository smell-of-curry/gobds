package session

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/df-mc/dragonfly/server/session"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
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

	afkTimer *infra.AFKTimer
	border   *area.Area2D

	claimPrefilter     bool
	claimDenyRendering bool

	close chan struct{}

	afk afkState

	corrective correctiveState
	traffic    trafficState

	// lastForwardedPing is the last client latency (ms) sent to BDS via
	// ForwardPing. -1 means nothing has been forwarded yet.
	lastForwardedPing int64

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

type correctiveState struct {
	mu   sync.Mutex
	last map[correctiveKey]time.Time
}

type correctiveKey struct {
	dimension int32
	x, z      int32
}

type malformedPacketError struct {
	reason string
}

func (e malformedPacketError) Error() string {
	return "malformed packet: " + e.reason
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

// Position returns the latest sampled player position.
func (s *Session) Position() mgl32.Vec3 {
	s.afk.mu.RLock()
	defer s.afk.mu.RUnlock()
	return s.afk.lastPosition
}

func (s *Session) allowCorrective(chunkPos protocol.ChunkPos, interval time.Duration) bool {
	key := correctiveKey{
		dimension: s.Data().Dimension(),
		x:         chunkPos.X(),
		z:         chunkPos.Z(),
	}
	now := time.Now()
	s.corrective.mu.Lock()
	defer s.corrective.mu.Unlock()
	if now.Sub(s.corrective.last[key]) < interval {
		return false
	}
	s.corrective.last[key] = now
	return true
}

func (s *Session) claimMessage(position mgl32.Vec3, message string) {
	chunkPos := protocol.ChunkPos{
		int32(math.Floor(float64(position.X()))) >> 4,
		int32(math.Floor(float64(position.Z()))) >> 4,
	}
	if s.allowCorrective(chunkPos, time.Second) {
		s.Message(message)
		if s.claimFactory != nil {
			s.claimFactory.Metrics().Correction(true)
		}
	} else if s.claimFactory != nil {
		s.claimFactory.Metrics().Correction(false)
	}
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

// Data ...
func (s *Session) Data() *Data {
	return s.data
}

// Ping returns the client's RakNet latency in milliseconds (half RTT).
func (s *Session) Ping() int64 {
	return s.client.Latency().Milliseconds()
}

// ForwardPing sends the real client↔proxy latency to BDS when it changes.
// BDS getPing() only sees proxy↔BDS (~0 on same box), so the behavior pack
// must receive this via chat and drive the PHUD indicator itself.
func (s *Session) ForwardPing() {
	ping := s.Ping()
	if ping == s.lastForwardedPing {
		return
	}
	s.lastForwardedPing = ping
	s.WriteToServer(&packet.Text{
		TextType:         packet.TextTypeChat,
		NeedsTranslation: false,
		SourceName:       s.ClientData().ThirdPartyName,
		Message:          fmt.Sprintf("[PROXY_PING] %d", ping),
		XUID:             s.IdentityData().XUID,
	})
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
func (s *Session) handlePacket(p packet.Packet, conn Conn) (send bool, err error) {
	handler, ok := s.handlers[p.ID()]
	if !ok {
		return true, nil
	}
	ctx := event.C(conn)
	defer func() {
		if recovered := recover(); recovered != nil {
			s.traffic.malformed(trafficHandler)
			s.log.Error("panic handling packet", "packet_id", p.ID(), "error", recovered)
			s.Disconnect("Malformed client packet.")
			send = false
			err = fmt.Errorf("panic handling packet %d: %v", p.ID(), recovered)
		}
	}()
	err = handler.Handle(s, p, ctx)
	if err == nil {
		return !ctx.Cancelled(), nil
	}
	s.log.Error("error handling packet", "packet_id", p.ID(), "error", err)
	var malformed malformedPacketError
	if errors.As(err, &malformed) {
		s.Disconnect("Malformed client packet.")
		return false, err
	}
	// Handler failures drop only this packet. Claim and subchunk handlers are
	// fail-open internally; unrelated backend failures must not disconnect sessions.
	return false, nil
}

// registerHandlers registers all packet handlers.
func (s *Session) registerHandlers() {
	s.handlers = map[uint32]packetHandler{
		packet.IDAddActor:             &AddActorHandler{},
		packet.IDAddPainting:          &AddPaintingHandler{},
		packet.IDAvailableCommands:    &AvailableCommandsHandler{},
		packet.IDChangeDimension:      &ChangeDimensionHandler{},
		packet.IDCommandRequest:       &CommandRequestHandler{},
		packet.IDInventoryTransaction: &InventoryTransactionHandler{},
		packet.IDItemRegistry:         &ItemRegistryHandler{},
		packet.IDItemStackRequest:     &ItemStackRequestHandler{},
		packet.IDLevelChunk:           &LevelChunkHandler{},
		packet.IDModalFormRequest:     &ModalFormRequestHandler{},
		packet.IDModalFormResponse:    &ModalFormResponseHandler{},
		packet.IDMoveActorAbsolute:    &MoveActorHandler{},
		packet.IDMoveActorDelta:       &MoveActorHandler{},
		packet.IDPlayerAuthInput:      NewPlayerAuthInputHandler(),
		packet.IDRemoveActor:          &RemoveActorHandler{},
		packet.IDSetActorData:         &SetActorDataHandler{},
		packet.IDSetPlayerGameType:    &SetPlayerGameTypeHandler{},
		packet.IDSubChunk:             &SubChunkHandler{},
		packet.IDText:                 &TextHandler{},
		packet.IDUpdateAbilities:      &UpdateAbilitiesHandler{},
		packet.IDUpdatePlayerGameType: &UpdatePlayerGameTypeHandler{},
	}
}

// WriteTrafficMetrics emits and resets this session's traffic metric delta.
func (s *Session) WriteTrafficMetrics(output io.Writer, server string, period time.Duration) {
	s.traffic.session.WriteDelta(output, server, s.IdentityData().XUID, period)
}
