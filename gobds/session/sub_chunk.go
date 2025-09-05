package session

import (
	"slices"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// SubChunkHandler ...
type SubChunkHandler struct{}

// Handle ...
func (*SubChunkHandler) Handle(s *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.SubChunk)
	s.Data().SetDimension(pkt.Dimension)

	if s.border == nil {
		return nil
	}

	pkt.SubChunkEntries = slices.DeleteFunc(pkt.SubChunkEntries, func(entry protocol.SubChunkEntry) bool {
		return !s.border.ChunkInside(protocol.ChunkPos{
			pkt.Position.X() + int32(entry.Offset[0]),
			pkt.Position.Z() + int32(entry.Offset[2]),
		})
	})
	return nil
}
