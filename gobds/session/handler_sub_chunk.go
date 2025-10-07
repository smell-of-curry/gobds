package session

import (
	"bytes"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// SubChunkHandler ...
type SubChunkHandler struct{}

// Handle ...
func (*SubChunkHandler) Handle(s *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.SubChunk)
	s.Data().SetDimension(pkt.Dimension)

	dimension, _ := world.DimensionByID(int(pkt.Dimension))
	virtualChunk := chunk.New(airRuntimeID, dimension.Range())

	var entries []protocol.SubChunkEntry
	for _, entry := range pkt.SubChunkEntries {
		if entry.Result != protocol.SubChunkResultSuccess {
			continue
		}

		chunkPos := protocol.ChunkPos{
			pkt.Position.X() + int32(entry.Offset[0]),
			pkt.Position.Z() + int32(entry.Offset[2]),
		}

		if s.border != nil && !s.border.ChunkInside(chunkPos) {
			continue
		}

		claim, exists := ClaimAtChunk(pkt.Dimension, chunkPos)
		if !exists {
			entries = append(entries, entry)
			continue
		}

		if ClaimActionPermitted(claim, s, ClaimActionRender, chunkPos) {
			entries = append(entries, entry)
			continue
		}

		var index byte
		buf := bytes.NewBuffer(entry.RawPayload)
		decodedEntry, err := decodeSubChunk(buf, virtualChunk, &index, chunk.NetworkEncoding)
		if err != nil {
			s.log.Error("decode subchunk entry", "error", err)
			continue
		}

		centerY := pkt.Position.Y()
		offsetY := int32(entry.Offset[1])

		sectionY := centerY + offsetY
		bottomSectionY := int32(dimension.Range().Min() >> 4)
		if sectionY == bottomSectionY {
			for z := uint8(0); z < 16; z++ {
				for x := uint8(0); x < 16; x++ {
					decodedEntry.SetBlock(x, 0, z, 0, denyRuntimeID)
				}
			}
		}

		virtualChunk.Sub()[index] = decodedEntry
		entry.RawPayload = chunk.EncodeSubChunk(virtualChunk, chunk.NetworkEncoding, int(index))

		entries = append(entries, entry)
	}

	pkt.SubChunkEntries = entries
	return nil
}

//go:linkname decodeSubChunk github.com/df-mc/dragonfly/server/world/chunk.decodeSubChunk
func decodeSubChunk(buf *bytes.Buffer, c *chunk.Chunk, index *byte, e chunk.Encoding) (*chunk.SubChunk, error)
