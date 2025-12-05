package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// UpdateAbilities ...
type UpdateAbilities struct{}

// Handle ...
func (*UpdateAbilities) Handle(s *Session, pk packet.Packet, ctx *Context) error {
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
	position := s.handlers[packet.IDPlayerAuthInput].(*PlayerAuthInputHandler).lastPosition
	s.WriteToClient(&packet.LevelChunk{
		Position:      protocol.ChunkPos{int32(position.X()) >> 4, int32(position.Z()) >> 4},
		SubChunkCount: protocol.SubChunkRequestModeLimited,
	})
	return nil
}
