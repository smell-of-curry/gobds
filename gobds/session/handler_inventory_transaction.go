package session

import (
	"slices"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
)

// InventoryTransactionHandler ...
type InventoryTransactionHandler struct{}

// Handle ...
func (h *InventoryTransactionHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.InventoryTransaction)

	h.handleInteraction(s, pkt, ctx)
	if ctx.Cancelled() {
		return nil
	}

	h.handleWorldBorder(s, pkt, ctx)
	if ctx.Cancelled() {
		return nil
	}

	h.handleClaims(s, pkt, ctx)
	return nil
}

// handleInteraction ...
func (h *InventoryTransactionHandler) handleInteraction(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	data := s.Data()
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
		if !data.InteractWithBlock() {
			ctx.Cancel()
			return
		}
	}
}

// handleWorldBorder ...
func (h *InventoryTransactionHandler) handleWorldBorder(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	if s.border == nil {
		return
	}

	switch td := pkt.TransactionData.(type) {
	case *protocol.UseItemTransactionData:
		if td.ActionType == protocol.UseItemActionClickBlock {
			if !s.border.PositionInside(td.BlockPosition.X(), td.BlockPosition.Z()) {
				ctx.Cancel()
			}
		}
	}
}

// handleClaims ...
func (h *InventoryTransactionHandler) handleClaims(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	h.handleClaimUseItem(s, pkt, ctx)
	if ctx.Cancelled() {
		return
	}
	h.handleClaimUseItemOnEntity(s, pkt, ctx)
	if ctx.Cancelled() {
		return
	}
	h.handleClaimReleaseItem(s, pkt, ctx)
}

// handleClaimUseItem ...
func (h *InventoryTransactionHandler) handleClaimUseItem(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.UseItemTransactionData)
	if !ok {
		return
	}

	clientXUID := s.IdentityData().XUID

	dat := s.Data()
	pos := transactionData.Position
	cl, ok := ClaimAt(dat.Dimension(), pos.X(), pos.Z())
	if !ok {
		return
	}

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
				s.Message(text.Colourf("<red>You cannot interact with block entities inside this claim.</red>"))
				ctx.Cancel()
			}
		}
	}

	heldItem := transactionData.HeldItem.Stack.ItemType
	if heldItem.NetworkID == 0 {
		return
	}

	registry := s.handlers[packet.IDItemRegistry].(*ItemRegistryHandler)
	if registry.items == nil {
		return
	}

	entry, ok := registry.items[int16(heldItem.NetworkID)]
	if !ok {
		return
	}

	components, ok := entry.Data["components"].(map[string]any)
	if !ok || components["minecraft:throwable"] == nil {
		return
	}

	s.Message(text.Colourf("<red>You cannot throw items inside this claim.</red>"))
	ctx.Cancel()
}

// handleClaimUseItemEntity ...
func (h *InventoryTransactionHandler) handleClaimUseItemOnEntity(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.UseItemOnEntityTransactionData)
	if !ok {
		return
	}

	clientXUID := s.IdentityData().XUID

	dat := s.Data()
	pos := transactionData.Position
	cl, ok := ClaimAt(dat.Dimension(), pos.X(), pos.Z())
	if !ok {
		return
	}

	if cl.ID == "" || // Invalid claim?
		cl.OwnerXUID == "*" || // Admin claim.
		cl.OwnerXUID == clientXUID ||
		slices.Contains(cl.TrustedXUIDS, clientXUID) {
		return
	}
	ent, ok := infra.EntityFactory.ByRuntimeID(transactionData.TargetEntityRuntimeID)
	if !ok {
		return
	}

	switch ent.ActorType() {
	case "minecraft:armor_stand", "minecraft:painting":
		s.Message(text.Colourf("<red>You cannot interact with block entities inside this claim.</red>"))
		ctx.Cancel()
	}
}

// handleClaimReleaseItem ...
func (h *InventoryTransactionHandler) handleClaimReleaseItem(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.ReleaseItemTransactionData)
	if !ok {
		return
	}

	clientXUID := s.IdentityData().XUID

	dat := s.Data()
	pos := transactionData.HeadPosition.Sub(mgl32.Vec3{0, 1.62})
	cl, ok := ClaimAt(dat.Dimension(), pos.X(), pos.Z())
	if !ok {
		return
	}

	if cl.ID == "" || // Invalid claim?
		cl.OwnerXUID == "*" || // Admin claim.
		cl.OwnerXUID == clientXUID ||
		slices.Contains(cl.TrustedXUIDS, clientXUID) {
		return
	}

	s.Message(text.Colourf("<red>You cannot release items inside this claim.</red>"))
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
func ClaimAt(dimension int32, x, z float32) (claim.PlayerClaim, bool) {
	for _, c := range infra.Claims() {
		if claimDimensionToInt(c.Location.Dimension) == dimension {
			minX := min(c.Location.Pos1.X, c.Location.Pos2.X)
			maxX := max(c.Location.Pos1.X, c.Location.Pos2.X)
			minZ := min(c.Location.Pos1.Z, c.Location.Pos2.Z)
			maxZ := max(c.Location.Pos1.Z, c.Location.Pos2.Z)
			if x >= minX && x <= maxX && z >= minZ && z <= maxZ {
				return c, true
			}
		}
	}
	return claim.PlayerClaim{}, false
}
