package session

import (
	"bytes"
	_ "unsafe"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/df-mc/dragonfly/server/world/chunk"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/claim"
)

// SubChunkHandler ...
type SubChunkHandler struct{}

// Handle ...
func (*SubChunkHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	if ctx.Val() != s.server {
		return nil
	}
	pkt := pk.(*packet.SubChunk)
	s.Data().SetDimension(pkt.Dimension)

	dimensions := s.GameData().Dimensions
	dimensionRange, dimensionFound := dimensionRangeByID(pkt.Dimension, dimensions)
	claims := s.claimFactory.All()
	denyID := denyBlockRuntimeID(s.GameData().UseBlockNetworkIDHashes)

	entries := make([]protocol.SubChunkEntry, 0, len(pkt.SubChunkEntries))
	for _, entry := range pkt.SubChunkEntries {
		chunkPos := protocol.ChunkPos{
			pkt.Position.X() + int32(entry.Offset[0]),
			pkt.Position.Z() + int32(entry.Offset[2]),
		}

		if s.border != nil && !s.border.ChunkInside(chunkPos) {
			continue
		}

		// GoBDS disables the backend blob cache, so successful entries normally
		// contain the sub-chunk bytes inline. Leave cached entries untouched if
		// that invariant changes.
		if entry.Result != protocol.SubChunkResultSuccess || !dimensionFound || pkt.CacheEnabled {
			entries = append(entries, entry)
			continue
		}

		entries = append(entries, applyClaimDenyBlocks(
			s,
			pkt,
			entry,
			chunkPos,
			dimensionRange,
			dimensions,
			claims,
			denyID,
		))
	}

	pkt.SubChunkEntries = entries
	return nil
}

func applyClaimDenyBlocks(
	s *Session,
	pkt *packet.SubChunk,
	entry protocol.SubChunkEntry,
	chunkPos protocol.ChunkPos,
	dimensionRange cube.Range,
	dimensions []protocol.DimensionDefinition,
	claims map[string]claim.PlayerClaim,
	denyID uint32,
) protocol.SubChunkEntry {
	sectionY := pkt.Position.Y() + int32(entry.Offset[1])
	if sectionY != int32(dimensionRange.Min()>>4) {
		return entry
	}
	claimsInChunk := claimsAtChunk(claims, pkt.Dimension, dimensions, chunkPos)
	if len(claimsInChunk) == 0 {
		return entry
	}

	virtualChunk := chunk.New(world.DefaultBlockRegistry, dimensionRange)
	var index byte
	buf := bytes.NewBuffer(entry.RawPayload)
	decodedEntry, err := decodeSubChunk(buf, virtualChunk, &index, chunk.NetworkEncoding)
	if err != nil {
		s.log.Error("decode subchunk entry", "error", err)
		return entry
	}
	if int(index) >= len(virtualChunk.Sub()) {
		s.log.Error("decode subchunk entry", "error", "subchunk index outside dimension range", "index", index)
		return entry
	}
	blockEntityPayload := bytes.Clone(buf.Bytes())

	var modified bool
	for z := uint8(0); z < 16; z++ {
		for x := uint8(0); x < 16; x++ {
			blockPos := protocol.BlockPos{
				(chunkPos.X() << 4) + int32(x), 0, (chunkPos.Z() << 4) + int32(z),
			}
			position := blockPosToVec3(blockPos)
			denied := false
			for _, cl := range claimsInChunk {
				if claimContains(cl, position.X(), position.Z()) &&
					!ClaimActionPermitted(cl, s, ClaimActionRender, position) {
					denied = true
					break
				}
			}
			if !denied {
				continue
			}
			decodedEntry.SetBlock(x, 0, z, 0, denyID)
			modified = true
		}
	}
	if !modified {
		return entry
	}
	virtualChunk.Sub()[index] = decodedEntry
	entry.RawPayload = append(
		chunk.EncodeSubChunk(virtualChunk, chunk.NetworkEncoding, int(index)),
		blockEntityPayload...,
	)
	return entry
}

// decodeSubChunk links Dragonfly's unexported single-subchunk decoder.
//
//go:linkname decodeSubChunk github.com/df-mc/dragonfly/server/world/chunk.decodeSubChunk
func decodeSubChunk(buf *bytes.Buffer, c *chunk.Chunk, index *byte, e chunk.Encoding) (*chunk.SubChunk, error)
