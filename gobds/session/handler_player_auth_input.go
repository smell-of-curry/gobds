package session

import (
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
)

// PlayerAuthInputHandler ...
type PlayerAuthInputHandler struct {
	lastMoveTime       time.Time
	lastPosition       mgl32.Vec3
	lastYaw, lastPitch float32
	mu                 sync.Mutex
}

// NewPlayerAuthInputHandler ...
func NewPlayerAuthInputHandler() *PlayerAuthInputHandler {
	return &PlayerAuthInputHandler{
		lastMoveTime: time.Now(),
		lastPosition: mgl32.Vec3{},
		lastYaw:      0,
		lastPitch:    0,
	}
}

// Handle ...
func (h *PlayerAuthInputHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.PlayerAuthInput)

	if pkt.Tick%20 == 0 {
		s.SendPingIndicator()
		h.handleAFKTimer(s, pkt, ctx)
		if ctx.Cancelled() {
			return nil
		}
	}

	h.handleWorldInteractions(s, pkt, ctx)
	return nil
}

// handleAFKTimer ...
func (h *PlayerAuthInputHandler) handleAFKTimer(s *Session, pkt *packet.PlayerAuthInput, ctx *Context) {
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

	if time.Since(h.lastMoveTime) > s.afkTimer.TimeoutDuration {
		s.Disconnect(text.Colourf("<red>You've been kicked for being AFK.</red>"))
		ctx.Cancel()
	}
}

// handleWorldInteractions ...
func (h *PlayerAuthInputHandler) handleWorldInteractions(s *Session, pkt *packet.PlayerAuthInput, ctx *Context) {
	clientData := s.Data()
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

		if s.border != nil && !s.border.PositionInside(blockPosition.X(), blockPosition.Z()) {
			ctx.Cancel()
			continue
		}

		claim, exists := ClaimAt(clientData.Dimension(), float32(blockPosition.X()), float32(blockPosition.Z()))
		if !exists {
			continue
		}
		permitted := ClaimActionPermitted(claim, s, ClaimActionBlockInteract, blockPosToVec3(blockPosition))
		if permitted {
			continue
		}

		if clientData.GameMode() == packet.GameTypeCreative {
			s.WriteToClient(&packet.LevelChunk{
				Position:      protocol.ChunkPos{blockPosition.X() >> 4, blockPosition.Z() >> 4},
				SubChunkCount: protocol.SubChunkRequestModeLimited,
			})
		}
		ctx.Cancel()
	}
}
