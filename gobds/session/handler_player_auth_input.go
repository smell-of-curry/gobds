package session

import (
	"slices"

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
		s.SendPingIndicator()
		if s.afkTimer != nil {
			s.TouchMovement(pkt.Position, pkt.Yaw, pkt.Pitch)
		}
	}

	h.handleWorldBorder(s, pkt, ctx)
	return nil
}

// handleWorldBorder ...
func (h *PlayerAuthInputHandler) handleWorldBorder(s *Session, pkt *packet.PlayerAuthInput, ctx *Context) {
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

		claim, exists := ClaimAt(s.claimFactory.All(), clientData.Dimension(), float32(blockPosition.X()), float32(blockPosition.Z()))
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
