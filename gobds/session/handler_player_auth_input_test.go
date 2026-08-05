package session

import (
	"testing"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestPlayerAuthInputFiltersOnlyDeniedBlockActions(t *testing.T) {
	denied := protocol.PlayerBlockAction{
		Action:   protocol.PlayerActionStartBreak,
		BlockPos: protocol.BlockPos{1, 2, 3},
	}
	allowed := protocol.PlayerBlockAction{
		Action:   protocol.PlayerActionContinueDestroyBlock,
		BlockPos: protocol.BlockPos{4, 5, 6},
	}
	unsupported := protocol.PlayerBlockAction{
		Action:   protocol.PlayerActionCrackBreak,
		BlockPos: protocol.BlockPos{7, 8, 9},
	}
	pkt := &packet.PlayerAuthInput{
		Position:               mgl32.Vec3{10, 20, 30},
		MoveVector:             mgl32.Vec2{0.5, -0.5},
		VehicleRotation:        protocol.Option(mgl32.Vec2{40, 50}),
		ClientPredictedVehicle: protocol.Option(int64(99)),
		BlockActions:           protocol.Option([]protocol.PlayerBlockAction{denied, allowed, unsupported}),
	}

	deniedPositions := filterPlayerAuthInputPacket(pkt, func(action protocol.PlayerBlockAction) bool {
		return action.BlockPos == denied.BlockPos
	})

	actions, ok := pkt.BlockActions.Value()
	if !ok || len(actions) != 2 || actions[0] != allowed || actions[1] != unsupported {
		t.Fatalf("unexpected forwarded actions: %+v", pkt.BlockActions)
	}
	if len(deniedPositions) != 1 || deniedPositions[0] != denied.BlockPos {
		t.Fatalf("unexpected denied positions: %+v", deniedPositions)
	}
	vehicleRotation, hasVehicleRotation := pkt.VehicleRotation.Value()
	clientPredictedVehicle, hasClientPredictedVehicle := pkt.ClientPredictedVehicle.Value()
	if pkt.Position != (mgl32.Vec3{10, 20, 30}) ||
		pkt.MoveVector != (mgl32.Vec2{0.5, -0.5}) ||
		!hasVehicleRotation || vehicleRotation != (mgl32.Vec2{40, 50}) ||
		!hasClientPredictedVehicle || clientPredictedVehicle != 99 {
		t.Fatal("movement or vehicle input changed while filtering block actions")
	}
}

func TestCorrectiveLevelChunkUsesCustomDimensionFields(t *testing.T) {
	definitions := []protocol.DimensionDefinition{{
		Name:          "pokeb:battle_arena",
		Range:         [2]int32{-64, 320},
		DimensionType: 1000,
	}}
	chunkPos := protocol.ChunkPos{12, -7}
	correction, ok := correctiveLevelChunk(chunkPos, 1000, definitions)
	if !ok {
		t.Fatal("custom dimension correction was not built")
	}
	subChunkLimit, hasSubChunkLimit := correction.SubChunkLimit.Value()
	if correction.Position != chunkPos ||
		correction.Dimension != 1000 ||
		!hasSubChunkLimit || subChunkLimit != 24 ||
		correction.SubChunkCount != protocol.SubChunkRequestModeLimited {
		t.Fatalf("unexpected custom dimension correction: %+v", correction)
	}
}

func TestCorrectiveLevelChunkFailsOpenForUnknownDimension(t *testing.T) {
	if correction, ok := correctiveLevelChunk(protocol.ChunkPos{}, 1000, nil); ok || correction != nil {
		t.Fatalf("unknown dimension produced correction: %+v", correction)
	}
}
