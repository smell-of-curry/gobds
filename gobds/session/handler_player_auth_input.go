package session

import (
	"time"

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
func (h *PlayerAuthInputHandler) Handle(s *Session, pk packet.Packet, _ *Context) (err error) {
	if s.claimFactory != nil {
		s.claimFactory.Metrics().Packet()
		start := time.Now()
		defer func() { s.claimFactory.Metrics().Latency(time.Since(start)) }()
	}
	pkt := pk.(*packet.PlayerAuthInput)
	originalActions := pkt.BlockActions
	defer func() {
		if recovered := recover(); recovered != nil {
			pkt.BlockActions = originalActions
			s.log.Error("claim player auth input failed open", "error", recovered)
		}
	}()

	if pkt.Tick%20 == 0 {
		s.ForwardPing()
		s.TouchMovement(pkt.Position, pkt.Yaw, pkt.Pitch)
	}

	h.handleWorldInteractions(s, pkt)
	return nil
}

// handleWorldInteractions ...
func (h *PlayerAuthInputHandler) handleWorldInteractions(s *Session, pkt *packet.PlayerAuthInput) {
	clientData := s.Data()
	denied := filterPlayerAuthInputPacket(pkt, func(blockAction protocol.PlayerBlockAction) bool {
		blockPosition := blockAction.BlockPos
		if s.border != nil && !s.border.PositionInside(blockPosition.X(), blockPosition.Z()) {
			return true
		}
		return !s.claimActionPermitted(ClaimActionBlockBreak, blockPosToVec3(blockPosition))
	})
	if clientData.GameMode() != packet.GameTypeCreative {
		return
	}
	for _, blockPosition := range denied {
		chunkPos := protocol.ChunkPos{blockPosition.X() >> 4, blockPosition.Z() >> 4}
		correction, ok := correctiveLevelChunk(
			chunkPos,
			clientData.Dimension(),
			s.GameData().Dimensions,
		)
		if !ok {
			s.claimFactory.Metrics().Correction(false)
			continue
		}
		if !s.allowCorrective(chunkPos, time.Second) {
			s.claimFactory.Metrics().Correction(false)
			continue
		}
		s.WriteToClient(correction)
		s.claimFactory.Metrics().Correction(true)
	}
}

func correctiveLevelChunk(
	chunkPos protocol.ChunkPos,
	dimension int32,
	definitions []protocol.DimensionDefinition,
) (*packet.LevelChunk, bool) {
	dimensionRange, ok := dimensionRangeByID(dimension, definitions)
	if !ok || dimensionRange.Height() <= 0 {
		return nil, false
	}
	subChunkCount := (dimensionRange.Height() + 15) >> 4
	if subChunkCount > int(^uint16(0)) {
		return nil, false
	}
	return &packet.LevelChunk{
		Position:        chunkPos,
		Dimension:       dimension,
		HighestSubChunk: uint16(subChunkCount),
		SubChunkCount:   protocol.SubChunkRequestModeLimited,
	}, true
}

func filterPlayerAuthInputPacket(
	pkt *packet.PlayerAuthInput,
	denied func(protocol.PlayerBlockAction) bool,
) []protocol.BlockPos {
	filtered := make([]protocol.PlayerBlockAction, 0, len(pkt.BlockActions))
	deniedPositions := make([]protocol.BlockPos, 0)
	for _, action := range pkt.BlockActions {
		if !isFilterableBlockAction(action.Action) || !denied(action) {
			filtered = append(filtered, action)
			continue
		}
		deniedPositions = append(deniedPositions, action.BlockPos)
	}
	pkt.BlockActions = filtered
	return deniedPositions
}

func isFilterableBlockAction(action int32) bool {
	switch action {
	case protocol.PlayerActionStartBreak,
		protocol.PlayerActionPredictDestroyBlock,
		protocol.PlayerActionContinueDestroyBlock:
		return true
	default:
		return false
	}
}
