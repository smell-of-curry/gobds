package session

import "github.com/sandertv/gophertunnel/minecraft/protocol/packet"

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
	s.Data().SetOperator(abilityData.PlayerPermissions == packet.PermissionLevelOperator)
	return nil
}
