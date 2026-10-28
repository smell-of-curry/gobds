package session

import (
	"math"
	"time"

	"github.com/df-mc/dragonfly/server/block"
	"github.com/df-mc/dragonfly/server/event"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/text"
	gblock "github.com/smell-of-curry/gobds/gobds/block"
	"github.com/smell-of-curry/gobds/gobds/entity"
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
		// Hold-to-interact re-fires every simulation tick. BEH's claimManager
		// cancels non-first playerInteractWithBlock events after paying for the
		// native isFirstEvent getter (~7% script self-time). Drop those packets
		// here so they never reach BDS — and before traffic.allow, so hold-spam
		// does not burn the inventory rate budget (RUBY exceeded=16k/session).
		if dropHoldInteract(pkt) {
			ctx.Cancel()
			return nil
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

// dropHoldInteract reports whether this is a ClickBlock hold-repeat that BEH
// would cancel via isFirstEvent=false. TriggerTypeSimulationTick is the
// protocol equivalent of that gate.
func dropHoldInteract(pkt *packet.InventoryTransaction) bool {
	transactionData, ok := pkt.TransactionData.(*protocol.UseItemTransactionData)
	if !ok {
		return false
	}
	return transactionData.ActionType == protocol.UseItemActionClickBlock &&
		transactionData.TriggerType == protocol.TriggerTypeSimulationTick
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
		windowID, hasWindowID := action.WindowID.Value()
		if action.SourceType != protocol.InventoryActionSourceWorld || !hasWindowID || windowID != protocol.WindowIDInventory {
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
		if s.claimMessageAllowed(position) {
			s.Message(text.Colourf("<red>You cannot drop items inside this claim.</red>"))
		}
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
		return
	}
	if transactionData.ActionType == protocol.UseItemActionUseAsAttack {
		h.handleClaimUseAsAttack(s, transactionData, ctx)
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
		if s.claimMessageAllowed(pos) {
			s.Message(text.Colourf("<red>You cannot place blocks inside this claim.</red>"))
		}
		ctx.Cancel()
		return
	}

	if s.claimActionPermitted(
		ClaimActionBlockInteract,
		claimActionData{position: pos, typeID: typeID},
	) {
		return
	}
	if s.claimMessageAllowed(pos) {
		s.Message(text.Colourf("<red>You cannot interact with blocks inside this claim.</red>"))
	}
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

// decorationAttackOperator ...
// Operators bypass decoration-attack claim checks only in creative, so a
// survival operator cannot punch spawn cushions, armor stands, or paintings.
func decorationAttackOperator(operator bool, gameMode int32) bool {
	return operator && gameMode == packet.GameTypeCreative
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

	// Only decoration entities are claim-filtered here (armor stands, paintings,
	// cushions). Widening this to every non-player actor would deny interacting
	// with pokémon, NPCs, boats and the like inside any claim, and the proxy
	// sees ~25 of these actions per 4h session, so there is nothing to win by
	// moving that decision here.
	actorType := entity.ActorType()
	action, actionName, claimed := decorationUseOnEntityAction(actorType, transactionData.ActionType)
	if !claimed {
		return
	}
	entityPosition := entity.Position()
	data := claimActionData{
		position: entityPosition,
		typeID:   actorType,
	}
	var permitted bool
	if action == ClaimActionEntityHurt || decorationHoldsItems(data) {
		op := decorationAttackOperator(s.Data().Operator(), s.Data().GameMode())
		permitted = s.claimActionPermittedOperator(action, data, &op)
	} else {
		permitted = s.claimActionPermitted(action, data)
	}
	if permitted {
		return
	}

	if s.claimMessageAllowed(entityPosition) {
		s.Message(text.Colourf("<red>You cannot %s entities inside this claim.</red>", actionName))
	}
	ctx.Cancel()
}

// useItemOnEntityActionItemInteract is ItemUseOnActorActionType value 2 on
// protocol 2225. gophertunnel names only Interact (0) and Attack (1).
const useItemOnEntityActionItemInteract int32 = 2

// Claim decoration actor types.
const (
	actorArmorStand = "minecraft:armor_stand"
	actorPainting   = "minecraft:painting"
	actorCushion    = "minecraft:cushion"
)

const (
	decorationSwingReach  float32 = 5
	decorationSwingRadius float32 = 1
)

// decorationUseOnEntityAction reports the claim action for a decoration
// UseItemOnEntity packet. Item-interact (held item) breaks paintings the same
// way a punch does, and is ignored for cushions so sitting stays open.
func decorationUseOnEntityAction(actorType string, actionType int32) (ClaimAction, string, bool) {
	switch actorType {
	case actorArmorStand, actorPainting, actorCushion:
	default:
		return 0, "", false
	}
	switch actionType {
	case protocol.UseItemOnEntityActionInteract:
		return ClaimActionEntityInteract, "interact with", true
	case protocol.UseItemOnEntityActionAttack:
		return ClaimActionEntityHurt, "hurt", true
	case useItemOnEntityActionItemInteract:
		if actorType != actorPainting {
			return 0, "", false
		}
		return ClaimActionEntityHurt, "hurt", true
	default:
		return 0, "", false
	}
}

// handleClaimUseAsAttack drops a melee swing that would hit a claim decoration.
// 1.26.60 sends that swing as UseItem action UseAsAttack, which has no entity
// runtime id; the server raycasts. UseItemOnEntity never sees it.
func (h *InventoryTransactionHandler) handleClaimUseAsAttack(
	s *Session,
	data *protocol.UseItemTransactionData,
	ctx *Context,
) {
	if s.entityFactory == nil {
		return
	}
	yaw, pitch := s.Look()
	// Bedrock player positions are already at eye height.
	origin := data.Position
	// ponytail: scans every actor the client has loaded, once per melee swing.
	// Upgrade path is a decoration-only index if this shows up in a profile.
	hit, ok := closestDecorationSwing(origin, lookDirection(yaw, pitch), s.entityFactory.All())
	if !ok {
		return
	}
	actionData := claimActionData{position: hit.Position(), typeID: hit.ActorType()}
	op := decorationAttackOperator(s.Data().Operator(), s.Data().GameMode())
	if s.claimActionPermittedOperator(ClaimActionEntityHurt, actionData, &op) {
		return
	}
	s.log.Info("dropped claim decoration swing", "type", hit.ActorType(), "position", hit.Position())
	if s.claimMessageAllowed(hit.Position()) {
		s.Message(text.Colourf("<red>You cannot hurt entities inside this claim.</red>"))
	}
	ctx.Cancel()
}

// lookDirection is the unit look vector for Bedrock yaw/pitch degrees.
// Yaw 0 looks +Z. Positive pitch looks down.
func lookDirection(yaw, pitch float32) mgl32.Vec3 {
	yawRad := float64(yaw) * math.Pi / 180
	pitchRad := float64(pitch) * math.Pi / 180
	m := math.Cos(pitchRad)
	return mgl32.Vec3{
		float32(-m * math.Sin(yawRad)),
		float32(-math.Sin(pitchRad)),
		float32(m * math.Cos(yawRad)),
	}
}

func decorationAimPoint(actor entity.Entity) mgl32.Vec3 {
	pos := actor.Position()
	switch actor.ActorType() {
	case actorArmorStand:
		pos[1] += 1
	case actorCushion:
		pos[1] += 0.3
	}
	return pos
}

// closestDecorationSwing returns the nearest claim decoration the swing ray hits.
func closestDecorationSwing(origin, dir mgl32.Vec3, actors []entity.Entity) (entity.Entity, bool) {
	var best entity.Entity
	found := false
	bestDist := decorationSwingReach
	for _, actor := range actors {
		switch actor.ActorType() {
		case actorArmorStand, actorPainting, actorCushion:
		default:
			continue
		}
		target := decorationAimPoint(actor)
		dist := target.Sub(origin).Dot(dir)
		if dist < 0 || dist > decorationSwingReach {
			continue
		}
		if origin.Add(dir.Mul(dist)).Sub(target).Len() > decorationSwingRadius {
			continue
		}
		if !found || dist < bestDist {
			best = actor
			bestDist = dist
			found = true
		}
	}
	return best, found
}
