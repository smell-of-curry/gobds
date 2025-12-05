package session

import (
	"github.com/df-mc/dragonfly/server/block"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/infra"
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
	b, ok := blockByRuntimeID(transactionData.BlockRuntimeID)
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

	if transaction, ok := pkt.TransactionData.(*protocol.UseItemTransactionData); ok {
		if transaction.ActionType == protocol.UseItemActionClickBlock {
			if !s.border.PositionInside(transaction.BlockPosition.X(), transaction.BlockPosition.Z()) {
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

	clientData := s.Data()
	pos := transactionData.ClickedPosition
	claim, exists := ClaimAt(clientData.Dimension(), pos.X(), pos.Z())
	if !exists {
		return
	}

	if transactionData.ActionType == protocol.UseItemActionClickBlock &&
		transactionData.TriggerType == protocol.UseItemActionClickAir {
		permitted := ClaimActionPermitted(claim, s, ClaimActionBlockInteract, pos)
		if permitted {
			return
		}
		if b, found := blockByRuntimeID(transactionData.BlockRuntimeID); found {
			switch b.(type) {
			case block.ItemFrame, block.Lectern, block.DecoratedPot:
				s.Message(text.Colourf("<red>You cannot interact with block entities inside this claim.</red>"))
				ctx.Cancel()
			}
		}
	}

	permitted := ClaimActionPermitted(claim, s, ClaimActionItemThrow, pos)
	if permitted {
		return
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

	clientData := s.Data()
	pos := transactionData.ClickedPosition
	claim, ok := ClaimAt(clientData.Dimension(), pos.X(), pos.Z())
	if !ok {
		return
	}
	permitted := ClaimActionPermitted(claim, s, ClaimActionBlockInteract, pos)
	if permitted {
		return
	}

	entity, ok := infra.EntityFactory.ByRuntimeID(transactionData.TargetEntityRuntimeID)
	if !ok {
		return
	}

	switch entity.ActorType() {
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

	clientData := s.Data()
	pos := transactionData.HeadPosition.Sub(mgl32.Vec3{0, 1.62})
	claim, ok := ClaimAt(clientData.Dimension(), pos.X(), pos.Z())
	if !ok {
		return
	}
	permitted := ClaimActionPermitted(claim, s, ClaimActionItemRelease, pos)
	if permitted {
		return
	}

	s.Message(text.Colourf("<red>You cannot release items inside this claim.</red>"))
	ctx.Cancel()
}
