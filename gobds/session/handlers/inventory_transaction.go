package handlers

import (
	"slices"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// InventoryTransaction ...
type InventoryTransaction struct{}

// Handle ...
func (h InventoryTransaction) Handle(c interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.InventoryTransaction)

	h.handleWorldBorder(c, pkt, ctx)
	if ctx.Cancelled() {
		return
	}

	h.handleClaims(c, pkt, ctx)
}

// handleWorldBorder ...
func (h InventoryTransaction) handleWorldBorder(_ interceptor.Client, pkt *packet.InventoryTransaction, ctx *session.Context) {
	if !infra.WorldBorderEnabled() {
		return
	}

	switch td := pkt.TransactionData.(type) {
	case *protocol.UseItemTransactionData:
		if td.ActionType == protocol.UseItemActionClickBlock {
			if !infra.WorldBorder.PositionInside(td.BlockPosition.X(), td.BlockPosition.Z()) {
				ctx.Cancel()
			}
		}
	}
}

// handleClaims ...
func (h InventoryTransaction) handleClaims(c interceptor.Client, pkt *packet.InventoryTransaction, ctx *session.Context) {
	td, ok := pkt.TransactionData.(*protocol.UseItemTransactionData)
	if ok {
		return
	}

	clientXUID := c.IdentityData().XUID

	dat, ok := c.Data().(interceptor.ClientData)
	if !ok {
		return
	}
	pos := td.Position
	cl := ClaimAt(dat.Dimension(), int32(pos.X()), int32(pos.Z()))

	if cl.ID == "" || // Invalid claim?
		cl.OwnerXUID == "*" || // Admin claim.
		cl.OwnerXUID == clientXUID ||
		slices.Contains(cl.TrustedXUIDS, clientXUID) {
		return
	}

	heldItem := td.HeldItem.Stack.ItemType
	if heldItem.NetworkID == 0 {
		return
	}

	itemsMu.RLock()
	var entry *protocol.ItemEntry
	for i := range itemsCache {
		if int16(heldItem.NetworkID) == itemsCache[i].RuntimeID {
			entry = &itemsCache[i]
			break
		}
	}
	itemsMu.RUnlock()

	if entry == nil {
		return
	}

	components, ok := entry.Data["components"].(map[string]any)
	if !ok || components["minecraft:throwable"] == nil {
		return
	}

	ctx.Cancel()
}

// claimDimensionToInt ...
func claimDimensionToInt(dimension string) int32 {
	if dimension == "minecraft:overworld" {
		return 0
	} else if dimension == "minecraft:nether" {
		return 1
	} else if dimension == "minecraft:end" {
		return 2
	} else {
		return -1
	}
}

// ClaimAt ...
func ClaimAt(dimension int32, x, z int32) claim.PlayerClaim {
	for _, c := range infra.Claims() {
		if claimDimensionToInt(c.Location.Dimension) == dimension {
			minX, minZ := int32(c.Location.MinPos.X), int32(c.Location.MinPos.Z)
			maxX, maxZ := int32(c.Location.MaxPos.X), int32(c.Location.MaxPos.Z)

			if x >= minX && x <= maxX {
				if z >= minZ && z <= maxZ {
					return c
				}
			}
		}
	}
	return claim.PlayerClaim{}
}
