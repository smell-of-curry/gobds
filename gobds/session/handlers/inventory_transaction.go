package handlers

import (
	"slices"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
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

	h.handleInteraction(c, pkt, ctx)
	if ctx.Cancelled() {
		return
	}

	h.handleWorldBorder(c, pkt, ctx)
	if ctx.Cancelled() {
		return
	}

	h.handleClaims(c, pkt, ctx)
}

// handleInteraction ...
func (h InventoryTransaction) handleInteraction(c interceptor.Client, pkt *packet.InventoryTransaction, ctx *session.Context) {
	data := c.Data().(interceptor.ClientData)
	for _, action := range pkt.Actions {
		if action.SourceType != protocol.InventoryActionSourceWorld || action.WindowID != protocol.WindowIDInventory {
			continue
		}
		data.SetLastDrop()
	}

	transactionData, ok := pkt.TransactionData.(*protocol.UseItemTransactionData)
	if !ok {
		return
	}
	if transactionData.ActionType != protocol.UseItemActionClickBlock &&
		transactionData.TriggerType != protocol.UseItemActionClickAir {
		return
	}
	b, ok := world.BlockByRuntimeID(transactionData.BlockRuntimeID)
	if !ok {
		return
	}
	switch b.(type) {
	case block.Chest, block.EnderChest, block.CraftingTable,
		block.Anvil, block.Stonecutter, block.Hopper,
		gblock.Button, block.WoodTrapdoor, block.CopperTrapdoor,
		block.WoodFenceGate, block.WoodDoor, block.CopperDoor,
		block.Ladder, block.Composter:
		ctx.Cancel()
		if !data.InteractWithBlock() {
			return
		}
		time.AfterFunc(time.Millisecond*500, func() {
			c.WriteToServer(pkt)
		})
	}
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
	h.handleClaimUseItem(c, pkt, ctx)
	if ctx.Cancelled() {
		return
	}
	h.handleClaimUseItemOnEntity(c, pkt, ctx)
	if ctx.Cancelled() {
		return
	}
	h.handleClaimReleaseItem(c, pkt, ctx)
}

// handleClaimUseItem ...
func (h InventoryTransaction) handleClaimUseItem(c interceptor.Client, pkt *packet.InventoryTransaction, ctx *session.Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.UseItemTransactionData)
	if !ok {
		return
	}

	clientXUID := c.IdentityData().XUID

	dat, ok := c.Data().(interceptor.ClientData)
	if !ok {
		return
	}
	pos := transactionData.Position
	cl := ClaimAt(dat.Dimension(), int32(pos.X()), int32(pos.Z()))

	if cl.ID == "" || // Invalid claim?
		cl.OwnerXUID == "*" || // Admin claim.
		cl.OwnerXUID == clientXUID ||
		slices.Contains(cl.TrustedXUIDS, clientXUID) {
		return
	}

	if transactionData.ActionType == protocol.UseItemActionClickBlock &&
		transactionData.TriggerType == protocol.UseItemActionClickAir {
		if b, exists := world.BlockByRuntimeID(transactionData.BlockRuntimeID); exists {
			switch b.(type) {
			case block.ItemFrame, block.Lectern, block.DecoratedPot:
				ctx.Cancel()
			}
		}
	}

	heldItem := transactionData.HeldItem.Stack.ItemType
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

// handleClaimUseItemEntity ...
func (h InventoryTransaction) handleClaimUseItemOnEntity(c interceptor.Client, pkt *packet.InventoryTransaction, ctx *session.Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.UseItemOnEntityTransactionData)
	if !ok {
		return
	}

	clientXUID := c.IdentityData().XUID

	dat, ok := c.Data().(interceptor.ClientData)
	if !ok {
		return
	}
	pos := transactionData.Position
	cl := ClaimAt(dat.Dimension(), int32(pos.X()), int32(pos.Z()))

	if cl.ID == "" || // Invalid claim?
		cl.OwnerXUID == "*" || // Admin claim.
		cl.OwnerXUID == clientXUID ||
		slices.Contains(cl.TrustedXUIDS, clientXUID) {
		return
	}
	ctx.Cancel()
}

// handleClaimReleaseItem ...
func (h InventoryTransaction) handleClaimReleaseItem(c interceptor.Client, pkt *packet.InventoryTransaction, ctx *session.Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.ReleaseItemTransactionData)
	if !ok {
		return
	}

	clientXUID := c.IdentityData().XUID

	dat, ok := c.Data().(interceptor.ClientData)
	if !ok {
		return
	}
	pos := transactionData.HeadPosition.Sub(mgl32.Vec3{0, 1.62})
	cl := ClaimAt(dat.Dimension(), int32(pos.X()), int32(pos.Z()))

	if cl.ID == "" || // Invalid claim?
		cl.OwnerXUID == "*" || // Admin claim.
		cl.OwnerXUID == clientXUID ||
		slices.Contains(cl.TrustedXUIDS, clientXUID) {
		return
	}
	ctx.Cancel()
}

// claimDimensionToInt ...
func claimDimensionToInt(dimension string) int32 {
	switch dimension {
	case "minecraft:overworld":
		return 0
	case "minecraft:nether":
		return 1
	case "minecraft:end":
		return 2
	default:
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
