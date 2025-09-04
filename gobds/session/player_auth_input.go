package session

import (
	"slices"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
)

// PlayerAuthInput ...
type PlayerAuthInput struct {
	lastMoveTime       time.Time
	lastPosition       mgl32.Vec3
	lastYaw, lastPitch float32
	mu                 sync.Mutex
}

// NewPlayerAuthInputHandler ...
func NewPlayerAuthInputHandler() *PlayerAuthInput {
	return &PlayerAuthInput{
		lastMoveTime: time.Now(),
		lastPosition: mgl32.Vec3{},
		lastYaw:      0,
		lastPitch:    0,
	}
}

// Handle ...
func (h *PlayerAuthInput) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.PlayerAuthInput)

	if pkt.Tick%20 == 0 {
		s.SendPingIndicator()
		h.handleAFKTimer(s, pkt, ctx)
		if ctx.Cancelled() {
			return nil
		}
	}

	h.handleWorldBorder(s, pkt, ctx)
	return nil
}

// handleAFKTimer ...
func (h *PlayerAuthInput) handleAFKTimer(s *Session, pkt *packet.PlayerAuthInput, ctx *Context) {
	if s.afkTimer == nil {
		return
	}

	moved := !h.lastPosition.ApproxEqual(pkt.Position) ||
		!mgl32.FloatEqual(h.lastYaw, pkt.Yaw) ||
		!mgl32.FloatEqual(h.lastPitch, pkt.Pitch)
	if moved {
		h.lastMoveTime = time.Now()
		h.lastPosition = pkt.Position
		h.lastYaw = pkt.Yaw
		h.lastPitch = pkt.Pitch
		return
	}

	if time.Since(h.lastMoveTime) > time.Duration(s.afkTimer.TimeoutDuration) {
		s.Disconnect(text.Colourf("<red>You've been kicked for being AFK.</red>"))
		ctx.Cancel()
	}
}

// handleWorldBorder ...
func (h *PlayerAuthInput) handleWorldBorder(s *Session, pkt *packet.PlayerAuthInput, ctx *Context) {
	clientData := s.Data()
	clientXUID := s.IdentityData().XUID
	for i, action := range pkt.BlockActions {
		if action.Action == protocol.PlayerActionCrackBreak {
			continue
		}

		blockPosition := action.BlockPos
		if action.Action == protocol.PlayerActionStopBreak && (i == 0 || i == 1) && len(pkt.BlockActions) > 1 {
			switch i {
			case 0:
				blockPosition = pkt.BlockActions[i+1].BlockPos
			default:
				blockPosition = pkt.BlockActions[i-1].BlockPos
			}
		}

		if s.border != nil && !s.border.PositionInside(blockPosition.X(), blockPosition.Z()) {
			ctx.Cancel()
			continue
		}

		claim, exists := ClaimAt(clientData.Dimension(), float32(blockPosition.X()), float32(blockPosition.Z()))
		if !exists {
			continue
		}
		if claim.ID == "" || // Invalid claim?
			claim.OwnerXUID == "*" || // Admin claim.
			claim.OwnerXUID == clientXUID ||
			slices.Contains(claim.TrustedXUIDS, clientXUID) {
			continue
		}

		ctx.Cancel()
	}
}
