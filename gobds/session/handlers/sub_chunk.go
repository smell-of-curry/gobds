package handlers

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// SubChunk ...
type SubChunk struct{}

// Handle ...
func (SubChunk) Handle(c interceptor.Client, pk packet.Packet, _ *session.Context) {
	pkt := pk.(*packet.SubChunk)

	if dat, ok := c.Data().(interceptor.ClientData); ok {
		dat.SetDimension(pkt.Dimension)
	}

	if !infra.WorldBorderEnabled() {
		return
	}

	var entries = make([]protocol.SubChunkEntry, 0)
	for _, subChunk := range pkt.SubChunkEntries {
		inside := infra.WorldBorder.ArePositionsInside([]int32{
			pkt.Position.X() + int32(subChunk.Offset[0]),
			pkt.Position.Z() + int32(subChunk.Offset[2]),
		})
		if inside {
			entries = append(entries, subChunk)
		}
	}

	pkt.SubChunkEntries = entries
}
