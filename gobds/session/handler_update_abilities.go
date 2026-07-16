package session

import (
	"math"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// UpdateAbilitiesHandler ...
type UpdateAbilitiesHandler struct{}

// Handle ...
func (*UpdateAbilitiesHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.UpdateAbilities)
	if ctx.Val() != s.server {
		return nil
	}
	abilityData := pkt.AbilityData
	if abilityData.EntityUniqueID != s.GameData().EntityUniqueID {
		return nil
	}

	prevOperator := s.Data().Operator()
	operator := abilityData.PlayerPermissions == packet.PermissionLevelOperator
	if prevOperator == operator {
		return nil
	}

	s.Data().SetOperator(operator)
	position := s.Position()
	chunkPos := protocol.ChunkPos{
		int32(math.Floor(float64(position.X()))) >> 4,
		int32(math.Floor(float64(position.Z()))) >> 4,
	}
	correction, ok := correctiveLevelChunk(chunkPos, s.Data().Dimension(), s.GameData().Dimensions)
	if !ok {
		return nil
	}
	s.WriteToClient(correction)
	return nil
}
