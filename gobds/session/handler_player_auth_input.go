package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// PlayerAuthInputHandler handles PlayerAuthInput packets. AFK state lives on
// the Session; this handler only feeds movement updates into it.
type PlayerAuthInputHandler struct{}

// NewPlayerAuthInputHandler ...
func NewPlayerAuthInputHandler() *PlayerAuthInputHandler {
	return &PlayerAuthInputHandler{}
}

// Handle ...
func (h *PlayerAuthInputHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.PlayerAuthInput)

	if pkt.Tick%20 == 0 {
		s.ForwardPing()
		s.TouchMovement(pkt.Position, pkt.Yaw, pkt.Pitch)
	}

	h.handleWorldInteractions(s, pkt, ctx)
	return nil
}

// handleWorldInteractions ...
func (h *PlayerAuthInputHandler) handleWorldInteractions(s *Session, pkt *packet.PlayerAuthInput, ctx *Context) {
	clientData := s.Data()
	for _, blockAction := range pkt.BlockActions {
		switch blockAction.Action {
		case protocol.PlayerActionStartBreak,
			protocol.PlayerActionPredictDestroyBlock,
			protocol.PlayerActionContinueDestroyBlock:
		default:
			continue
		}

		blockPosition := blockAction.BlockPos
		if s.border != nil && !s.border.PositionInside(blockPosition.X(), blockPosition.Z()) {
			ctx.Cancel()
			return
		}

		claim, exists := ClaimAt(
			s.claimFactory.All(),
			clientData.Dimension(),
			s.GameData().Dimensions,
			float32(blockPosition.X()),
			float32(blockPosition.Z()),
		)
		if !exists {
			continue
		}
		permitted := ClaimActionPermitted(claim, s, ClaimActionBlockBreak, blockPosToVec3(blockPosition))
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
		return
	}
}
