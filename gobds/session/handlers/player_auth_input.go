package handlers

import (
	"slices"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// PlayerAuthInput ...
type PlayerAuthInput struct{}

// Handle ...
func (h PlayerAuthInput) Handle(c interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.PlayerAuthInput)

	if pkt.Tick%20 == 0 {
		c.SendPingIndicator()
	}

	h.handleWorldBorder(c, pkt, ctx)
	if ctx.Cancelled() {
		return
	}

	h.handleClaims(c, pkt, ctx)
}

// handleWorldBorder ...
func (h PlayerAuthInput) handleWorldBorder(_ interceptor.Client, pkt *packet.PlayerAuthInput, ctx *session.Context) {
	if !infra.WorldBorderEnabled() {
		return
	}

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

		if !infra.WorldBorder.PositionInside(blockPosition.X(), blockPosition.Z()) {
			ctx.Cancel()
		}
	}
}

// handleClaims ...
func (h PlayerAuthInput) handleClaims(c interceptor.Client, pkt *packet.PlayerAuthInput, ctx *session.Context) {
	clientXUID := c.IdentityData().XUID
	for _, action := range pkt.BlockActions {
		dat, ok := c.Data().(interceptor.ClientData)
		if !ok {
			return
		}
		pos := action.BlockPos
		cl := ClaimAt(dat.Dimension(), pos.X(), pos.Z())

		if cl.ID == "" || // Invalid claim?
			cl.OwnerXUID == "*" || // Admin claim.
			cl.OwnerXUID == clientXUID ||
			slices.Contains(cl.TrustedXUIDS, clientXUID) {
			continue
		}

		ctx.Cancel()
	}
}
