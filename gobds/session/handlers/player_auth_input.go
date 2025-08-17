package handlers

import (
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// PlayerAuthInput ...
type PlayerAuthInput struct {
	lastMoveTime       time.Time
	lastPosition       mgl32.Vec3
	lastYaw, lastPitch float32
	mu                 sync.Mutex
}

// NewPlayerAuthInput ...
func NewPlayerAuthInput() *PlayerAuthInput {
	return &PlayerAuthInput{
		lastMoveTime: time.Now(),
		lastPosition: mgl32.Vec3{},
		lastYaw:      0,
		lastPitch:    0,
	}
}

// Handle ...
func (h *PlayerAuthInput) Handle(c interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.PlayerAuthInput)

	if pkt.Tick%20 == 0 {
		h.handleAFKTimer(c, pkt, ctx)
		if ctx.Cancelled() {
			return
		}
		c.SendPingIndicator()
	}

	h.handleWorldBorder(c, pkt, ctx)
	if ctx.Cancelled() {
		return
	}
}

// handleAFKTimer ...
func (h *PlayerAuthInput) handleAFKTimer(c interceptor.Client, pkt *packet.PlayerAuthInput, ctx *session.Context) {
	afkTimer := infra.AFKTimer
	if !afkTimer.Enabled {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

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

	if time.Since(h.lastMoveTime) > time.Duration(afkTimer.TimeoutDuration) {
		c.Disconnect(text.Colourf("<red>You've been kicked for being AFK.</red>"))
		ctx.Cancel()
	}
}

// handleWorldBorder ...
func (h *PlayerAuthInput) handleWorldBorder(c interceptor.Client, pkt *packet.PlayerAuthInput, ctx *session.Context) {
	clientData := c.Data().(interceptor.ClientData)
	for i, blockAction := range pkt.BlockActions {
		if blockAction.Action == protocol.PlayerActionCrackBreak {
			continue
		}

		blockPosition := blockAction.BlockPos
		if blockAction.Action == protocol.PlayerActionStopBreak && (i == 0 || i == 1) && len(pkt.BlockActions) > 1 {
			switch i {
			case 0:
				blockPosition = pkt.BlockActions[i+1].BlockPos
			default:
				blockPosition = pkt.BlockActions[i-1].BlockPos
			}
		}

		if infra.WorldBorderEnabled() &&
			!infra.WorldBorder.PositionInside(blockPosition.X(), blockPosition.Z()) {
			ctx.Cancel()
			continue
		}

		claim, exists := ClaimAt(clientData.Dimension(), float32(blockPosition.X()), float32(blockPosition.Z()))
		if !exists {
			continue
		}
		permitted := ClaimActionPermitted(claim, c, ClaimActionBlockInteract, blockPosToVec3(blockPosition))
		if permitted {
			continue
		}

		ctx.Cancel()
	}
}
