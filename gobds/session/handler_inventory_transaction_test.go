package session

import (
	"errors"
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/entity"
)

func TestDroppedItemNetworkIDOnlyAcceptsWorldActions(t *testing.T) {
	item := protocol.ItemInstance{
		Stack: protocol.ItemStack{ItemType: protocol.ItemType{NetworkID: 42}},
	}
	if id, ok := droppedItemNetworkID(protocol.InventoryAction{
		SourceType: protocol.InventoryActionSourceWorld,
		NewItem:    item,
	}); !ok || id != 42 {
		t.Fatalf("world drop returned id=%d, ok=%v", id, ok)
	}
	if id, ok := droppedItemNetworkID(protocol.InventoryAction{
		SourceType: protocol.InventoryActionSourceContainer,
		NewItem:    item,
	}); ok || id != 0 {
		t.Fatalf("container action treated as drop: id=%d, ok=%v", id, ok)
	}
	if id, ok := droppedItemNetworkID(protocol.InventoryAction{
		SourceType: protocol.InventoryActionSourceWorld,
	}); ok || id != 0 {
		t.Fatalf("empty world action treated as drop: id=%d, ok=%v", id, ok)
	}
}

func TestDecorationUseOnEntityAction(t *testing.T) {
	hurt, name, ok := decorationUseOnEntityAction("minecraft:painting", protocol.UseItemOnEntityActionAttack)
	if !ok || hurt != ClaimActionEntityHurt || name != "hurt" {
		t.Fatalf("painting punch = %v %q ok=%v", hurt, name, ok)
	}
	hurt, _, ok = decorationUseOnEntityAction("minecraft:painting", useItemOnEntityActionItemInteract)
	if !ok || hurt != ClaimActionEntityHurt {
		t.Fatalf("painting item interact = %v ok=%v", hurt, ok)
	}
	if _, _, ok = decorationUseOnEntityAction("minecraft:cushion", useItemOnEntityActionItemInteract); ok {
		t.Fatal("cushion item interact must stay unfiltered so sitting still works")
	}
	if _, _, ok = decorationUseOnEntityAction("minecraft:pig", protocol.UseItemOnEntityActionAttack); ok {
		t.Fatal("combat entities must not be claim-filtered here")
	}
}

func TestClosestDecorationSwingHitsPaintingInFront(t *testing.T) {
	origin := mgl32.Vec3{0, 1.62, 0}
	dir := lookDirection(0, 0)
	if dir[2] < 0.9 {
		t.Fatalf("yaw 0 should look +Z, got %v", dir)
	}
	painting := entity.NewEntity(1, 2, "minecraft:painting", mgl32.Vec3{0, 1.62, 3})
	behind := entity.NewEntity(3, 4, "minecraft:painting", mgl32.Vec3{0, 1.62, -2})
	pig := entity.NewEntity(5, 6, "minecraft:pig", mgl32.Vec3{0, 1.62, 2})
	side := entity.NewEntity(7, 8, "minecraft:cushion", mgl32.Vec3{4, 1.62, 3})
	hit, ok := closestDecorationSwing(origin, dir, []entity.Entity{behind, pig, side, painting})
	if !ok || hit.RuntimeID() != painting.RuntimeID() {
		t.Fatalf("swing hit runtime=%d ok=%v", hit.RuntimeID(), ok)
	}
	if _, ok = closestDecorationSwing(origin, dir, []entity.Entity{behind, pig, side}); ok {
		t.Fatal("swing must miss decorations that are behind or beside the look ray")
	}
	far := entity.NewEntity(9, 10, "minecraft:painting", mgl32.Vec3{0, 1.62, 8})
	if _, ok = closestDecorationSwing(origin, dir, []entity.Entity{far}); ok {
		t.Fatal("swing must miss a painting past reach")
	}
}

func TestOffsetBlockPosUsesBedrockFaceOrder(t *testing.T) {
	start := protocol.BlockPos{10, 20, 30}
	tests := []struct {
		face int32
		want protocol.BlockPos
	}{
		{face: 0, want: protocol.BlockPos{10, 19, 30}},
		{face: 1, want: protocol.BlockPos{10, 21, 30}},
		{face: 2, want: protocol.BlockPos{10, 20, 29}},
		{face: 3, want: protocol.BlockPos{10, 20, 31}},
		{face: 4, want: protocol.BlockPos{9, 20, 30}},
		{face: 5, want: protocol.BlockPos{11, 20, 30}},
	}
	for _, test := range tests {
		got, ok := offsetBlockPos(start, test.face)
		if !ok || got != test.want {
			t.Fatalf("face %d returned %v, ok=%v; want %v", test.face, got, ok, test.want)
		}
	}
	if _, ok := offsetBlockPos(start, 6); ok {
		t.Fatal("unknown face must fail open")
	}
}

func TestItemStackRequestActionBounds(t *testing.T) {
	config := DefaultTrafficConfig()
	dropTests := []struct {
		name string
		pkt  packet.ItemStackRequest
	}{
		{name: "empty"},
		{
			name: "empty actions",
			pkt: packet.ItemStackRequest{Requests: []protocol.ItemStackRequest{
				{RequestID: 1},
			}},
		},
	}
	for _, test := range dropTests {
		t.Run(test.name, func(t *testing.T) {
			drop, err := validateItemStackRequest(&test.pkt, config)
			if err != nil || !drop {
				t.Fatalf("expected quiet malformed drop, got drop=%v err=%v", drop, err)
			}
		})
	}
	disconnectTests := []struct {
		name string
		pkt  packet.ItemStackRequest
	}{
		{
			name: "too many requests",
			pkt: packet.ItemStackRequest{Requests: make(
				[]protocol.ItemStackRequest,
				config.MaxStackRequests+1,
			)},
		},
		{
			name: "too many actions",
			pkt: packet.ItemStackRequest{Requests: []protocol.ItemStackRequest{{
				RequestID: 1,
				Actions:   make([]protocol.StackRequestAction, config.MaxStackActions+1),
			}}},
		},
	}
	for _, test := range disconnectTests {
		t.Run(test.name, func(t *testing.T) {
			var malformed malformedPacketError
			drop, err := validateItemStackRequest(&test.pkt, config)
			if drop || !errors.As(err, &malformed) {
				t.Fatalf("expected malformed error, got %v", err)
			}
		})
	}
	valid := packet.ItemStackRequest{Requests: []protocol.ItemStackRequest{{
		RequestID: 1,
		Actions:   []protocol.StackRequestAction{nil},
	}}}
	if drop, err := validateItemStackRequest(&valid, config); err != nil || drop {
		t.Fatalf("valid bounded request rejected: drop=%v err=%v", drop, err)
	}
}
