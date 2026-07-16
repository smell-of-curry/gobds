package session

import (
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
)

// InventoryTransactionHandler ...
type InventoryTransactionHandler struct{}

// Handle ...
func (h *InventoryTransactionHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.InventoryTransaction)
	if ctx.Val() == s.client {
		if len(pkt.Actions) > s.traffic.config.MaxInventoryActions {
			s.traffic.malformed(trafficInventory)
			return malformedPacketError{reason: "inventory transaction has too many actions"}
		}
		if !s.traffic.allow(trafficInventory) {
			ctx.Cancel()
			return nil
		}
	}
	if s.claimFactory != nil {
		s.claimFactory.Metrics().Packet()
		start := time.Now()
		defer func() { s.claimFactory.Metrics().Latency(time.Since(start)) }()
	}

	h.handleInteraction(s, pkt, ctx)
	if ctx.Cancelled() {
		return nil
	}

	h.handleWorldBorder(s, pkt, ctx)
	if ctx.Cancelled() {
		return nil
	}

	if h.handleClaimsFailOpen(s, pkt, ctx.Val()) {
		ctx.Cancel()
	}
	return nil
}

func (h *InventoryTransactionHandler) handleClaimsFailOpen(
	s *Session,
	pkt *packet.InventoryTransaction,
	conn Conn,
) (cancel bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			cancel = false
			s.log.Error("claim inventory transaction failed open", "error", recovered)
		}
	}()
	ctx := event.C(conn)
	h.handleClaims(s, pkt, ctx)
	return ctx.Cancelled()
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
	if transactionData.ActionType != protocol.UseItemActionClickBlock ||
		transactionData.TriggerType != protocol.TriggerTypePlayerInput {
		return
	}
	b, ok := blockByRuntimeID(transactionData.BlockRuntimeID, s.GameData().UseBlockNetworkIDHashes)
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
	h.handleClaimDropItems(s, pkt, ctx)
	if ctx.Cancelled() {
		return
	}
	h.handleClaimUseItem(s, pkt, ctx)
	if ctx.Cancelled() {
		return
	}
	h.handleClaimUseItemOnEntity(s, pkt, ctx)
}

func (*InventoryTransactionHandler) handleClaimDropItems(
	s *Session,
	pkt *packet.InventoryTransaction,
	ctx *Context,
) {
	position := s.Position()
	registry := s.handlers[packet.IDItemRegistry].(*ItemRegistryHandler)
	for _, action := range pkt.Actions {
		networkID, ok := droppedItemNetworkID(action)
		if !ok {
			continue
		}
		entry, ok := registry.Item(int16(networkID))
		if !ok {
			continue
		}
		if s.claimActionPermitted(
			ClaimActionItemDrop,
			claimActionData{position: position, typeID: entry.Name},
		) {
			continue
		}
		s.claimMessage(position, text.Colourf("<red>You cannot drop items inside this claim.</red>"))
		ctx.Cancel()
		return
	}
}

func droppedItemNetworkID(action protocol.InventoryAction) (int32, bool) {
	if action.SourceType != protocol.InventoryActionSourceWorld {
		return 0, false
	}
	networkID := action.NewItem.Stack.NetworkID
	if networkID == 0 {
		networkID = action.OldItem.Stack.NetworkID
	}
	return networkID, networkID != 0
}

// handleClaimUseItem ...
func (h *InventoryTransactionHandler) handleClaimUseItem(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.UseItemTransactionData)
	if !ok {
		return
	}

	if transactionData.ActionType == protocol.UseItemActionClickBlock {
		h.handleClaimClickBlock(s, transactionData, ctx)
	}
}

func (*InventoryTransactionHandler) handleClaimClickBlock(
	s *Session,
	transactionData *protocol.UseItemTransactionData,
	ctx *Context,
) {
	pos := blockPosToVec3(transactionData.BlockPosition)
	b, found := blockByRuntimeID(transactionData.BlockRuntimeID, s.GameData().UseBlockNetworkIDHashes)
	if !found {
		return
	}
	typeID, _ := b.EncodeBlock()

	registry := s.handlers[packet.IDItemRegistry].(*ItemRegistryHandler)
	heldItem, heldItemFound := registry.Item(int16(transactionData.HeldItem.Stack.NetworkID))
	if heldItemFound && (heldItem.Name == "minecraft:stick" || heldItem.Name == "minecraft:golden_shovel") {
		return
	}

	if transactionData.HeldItem.Stack.BlockRuntimeID != 0 {
		placePos, validFace := offsetBlockPos(transactionData.BlockPosition, transactionData.BlockFace)
		placedBlock, placedBlockFound := blockByRuntimeID(
			uint32(transactionData.HeldItem.Stack.BlockRuntimeID),
			s.GameData().UseBlockNetworkIDHashes,
		)
		if !validFace || !placedBlockFound {
			return
		}
		placedTypeID, _ := placedBlock.EncodeBlock()
		if s.claimActionPermitted(
			ClaimActionBlockPlace,
			claimActionData{position: blockPosToVec3(placePos), typeID: placedTypeID},
		) {
			return
		}
		s.claimMessage(pos, text.Colourf("<red>You cannot place blocks inside this claim.</red>"))
		ctx.Cancel()
		return
	}

	if s.claimActionPermitted(
		ClaimActionBlockInteract,
		claimActionData{position: pos, typeID: typeID},
	) {
		return
	}
	s.claimMessage(pos, text.Colourf("<red>You cannot interact with blocks inside this claim.</red>"))
	ctx.Cancel()
}

func offsetBlockPos(pos protocol.BlockPos, face int32) (protocol.BlockPos, bool) {
	switch face {
	case 0:
		pos[1]--
	case 1:
		pos[1]++
	case 2:
		pos[2]--
	case 3:
		pos[2]++
	case 4:
		pos[0]--
	case 5:
		pos[0]++
	default:
		return pos, false
	}
	return pos, true
}

// handleClaimUseItemEntity ...
func (h *InventoryTransactionHandler) handleClaimUseItemOnEntity(s *Session, pkt *packet.InventoryTransaction, ctx *Context) {
	transactionData, ok := pkt.TransactionData.(*protocol.UseItemOnEntityTransactionData)
	if !ok {
		return
	}

	entity, exists := s.entityFactory.ByRuntimeID(transactionData.TargetEntityRuntimeID)
	if !exists {
		return
	}

	switch entity.ActorType() {
	case "minecraft:armor_stand", "minecraft:painting":
	default:
		return
	}
	entityPosition := entity.Position()

	var action ClaimAction
	var actionName string
	switch transactionData.ActionType {
	case protocol.UseItemOnEntityActionInteract:
		action = ClaimActionEntityInteract
		actionName = "interact with"
	case protocol.UseItemOnEntityActionAttack:
		action = ClaimActionEntityHurt
		actionName = "hurt"
	default:
		return
	}
	permitted := s.claimActionPermitted(action, claimActionData{
		position: entityPosition,
		typeID:   entity.ActorType(),
	})
	if permitted {
		return
	}

	s.claimMessage(entityPosition, text.Colourf("<red>You cannot %s entities inside this claim.</red>", actionName))
	ctx.Cancel()
}
