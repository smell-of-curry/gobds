package gobds

import (
	"testing"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestVoxelRegistryBelongsToJoiningConnection(t *testing.T) {
	p := &voxelProtocol{Protocol: minecraft.DefaultProtocol}
	one, two := new(minecraft.Conn), new(minecraft.Conn)
	shapes := &packet.VoxelShapes{
		CustomShapeCount: 1,
		NameMap:          []protocol.VoxelShapeNameEntry{{Name: "pokeb:test", ID: 2}},
	}
	p.registries.Store(one, shapes)
	placeholder := &packet.VoxelShapes{}
	for _, tc := range []struct {
		client *minecraft.Conn
		want   *packet.VoxelShapes
	}{{two, placeholder}, {one, shapes}, {one, placeholder}} {
		packets := p.ConvertFromLatest(placeholder, tc.client)
		if len(packets) != 1 || packets[0] != tc.want {
			t.Fatal("unexpected registry for joining connection")
		}
	}
}
