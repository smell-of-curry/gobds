package session

import (
	"errors"
	"testing"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
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
