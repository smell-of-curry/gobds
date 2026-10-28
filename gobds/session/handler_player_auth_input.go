package session

import (
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
	"github.com/smell-of-curry/gobds/gobds/entity"
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
	h.dropClaimDecorationSwing(s, pkt)
	return nil
}

// dropClaimDecorationSwing strips a melee swing that would hit a claim
// decoration. 1.26.60 sends that swing inside this packet, with no entity
// runtime id. The movement packet itself still goes through.
func (h *PlayerAuthInputHandler) dropClaimDecorationSwing(s *Session, pkt *packet.PlayerAuthInput) {
	if s.entityFactory == nil {
		return
	}
	hit, ok := stripClaimDecorationSwing(pkt, s.entityFactory.All(), func(e entity.Entity) bool {
		actionData := claimActionData{position: e.Position(), typeID: e.ActorType()}
		op := decorationAttackOperator(s.Data().Operator(), s.Data().GameMode())
		return !s.claimActionPermittedOperator(ClaimActionEntityHurt, actionData, &op)
	})
	if !ok {
		return
	}
	s.log.Info("dropped claim decoration swing", "type", hit.ActorType(), "position", hit.Position())
	if s.claimMessageAllowed(hit.Position()) {
		s.Message(text.Colourf("<red>You cannot hurt entities inside this claim.</red>"))
	}
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
		// Rate-limit before building/sending — suppression is ~90% of candidates.
		if !s.allowCorrective(chunkPos, time.Second) {
			if s.claimFactory != nil {
				s.claimFactory.Metrics().Correction(false)
			}
			continue
		}
		correction, ok := correctiveLevelChunk(
			chunkPos,
			clientData.Dimension(),
			s.GameData().Dimensions,
		)
		if !ok {
			if s.claimFactory != nil {
				s.claimFactory.Metrics().Correction(false)
			}
			continue
		}
		s.WriteToClient(correction)
		if s.claimFactory != nil {
			s.claimFactory.Metrics().Correction(true)
		}
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
	// Protocol 2168 dropped the MaxUint32 request-mode sentinels for
	// SubChunkCount (clients treat the field as a 0..64 count). Request mode
	// with a height limit is now SubChunkCount=0 plus SubChunkLimit set.
	return &packet.LevelChunk{
		Position:      chunkPos,
		Dimension:     dimension,
		SubChunkCount: 0,
		SubChunkLimit: protocol.Option(int32(subChunkCount)),
	}, true
}

func filterPlayerAuthInputPacket(
	pkt *packet.PlayerAuthInput,
	denied func(protocol.PlayerBlockAction) bool,
) []protocol.BlockPos {
	actions, ok := pkt.BlockActions.Value()
	if !ok {
		return nil
	}
	filtered := make([]protocol.PlayerBlockAction, 0, len(actions))
	deniedPositions := make([]protocol.BlockPos, 0)
	for _, action := range actions {
		if !isFilterableBlockAction(action.Action) || !denied(action) {
			filtered = append(filtered, action)
			continue
		}
		deniedPositions = append(deniedPositions, action.BlockPos)
	}
	pkt.BlockActions = protocol.Option(filtered)
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
