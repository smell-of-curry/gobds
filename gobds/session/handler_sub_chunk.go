package session

import (
	"bytes"
	"time"
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
func (*SubChunkHandler) Handle(s *Session, pk packet.Packet, ctx *Context) (err error) {
	if s.claimFactory != nil {
		s.claimFactory.Metrics().Packet()
		start := time.Now()
		defer func() { s.claimFactory.Metrics().Latency(time.Since(start)) }()
	}
	if ctx.Val() != s.server {
		return nil
	}
	pkt := pk.(*packet.SubChunk)
	originalEntries := pkt.SubChunkEntries
	defer func() {
		if recovered := recover(); recovered != nil {
			pkt.SubChunkEntries = originalEntries
			s.log.Error("claim subchunk handler failed open", "error", recovered)
		}
	}()
	s.Data().SetDimension(pkt.Dimension)

	dimensions := s.GameData().Dimensions
	var snapshot *claim.Snapshot
	var snapshotStatus claim.QueryStatus
	var dimension string
	var dimensionFound bool
	if s.claimDenyRendering && s.claimFactory != nil {
		snapshot, snapshotStatus = s.claimFactory.Snapshot(time.Now())
		dimension, dimensionFound = claimDimensionFromInt(pkt.Dimension, dimensions)
		if snapshotStatus != claim.QueryReady {
			s.claimFactory.Metrics().Reason(snapshotStatus)
		}
		if !dimensionFound {
			s.claimFactory.Metrics().Reason(claim.QueryUnknownDimension)
		}
	}

	entries := make([]protocol.SubChunkEntry, 0, len(pkt.SubChunkEntries))
	for _, entry := range pkt.SubChunkEntries {
		chunkPos := protocol.ChunkPos{
			pkt.Position.X() + int32(entry.Offset[0]),
			pkt.Position.Z() + int32(entry.Offset[2]),
		}

		if s.border != nil && !s.border.ChunkInside(chunkPos) {
			continue
		}

		if !s.claimDenyRendering || s.claimFactory == nil {
			entries = append(entries, entry)
			continue
		}

		dimensionRange, rangeFound := dimensionRangeByID(pkt.Dimension, dimensions)
		// GoBDS disables the backend blob cache, so successful entries normally
		// contain the sub-chunk bytes inline. Leave cached entries untouched if
		// that invariant changes.
		if entry.Result != protocol.SubChunkResultSuccess || !rangeFound || !dimensionFound ||
			snapshotStatus != claim.QueryReady || pkt.CacheEnabled {
			entries = append(entries, entry)
			continue
		}
		chunkX := float32(chunkPos.X() << 4)
		chunkZ := float32(chunkPos.Z() << 4)
		candidates := snapshot.Candidates(dimension, chunkX, chunkZ)
		s.claimFactory.Metrics().Candidates(len(candidates))

		entries = append(entries, applyClaimDenyBlocks(
			s,
			pkt,
			entry,
			chunkPos,
			dimensionRange,
			candidates,
			denyBlockRuntimeID(s.GameData().UseBlockNetworkIDHashes),
			ClaimActor{XUID: s.IdentityData().XUID, Operator: s.Data().Operator()},
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
	claims []*claim.PlayerClaim,
	denyID uint32,
	actor ClaimActor,
) protocol.SubChunkEntry {
	sectionY := pkt.Position.Y() + int32(entry.Offset[1])
	if sectionY != int32(dimensionRange.Min()>>4) {
		return entry
	}
	if len(claims) == 0 {
		return entry
	}

	virtualChunk := chunk.New(world.DefaultBlockRegistry, dimensionRange)
	var index byte
	buf := bytes.NewBuffer(entry.RawPayload)
	decodedEntry, err := decodeSubChunk(buf, virtualChunk, &index, chunk.NetworkEncoding)
	if err != nil {
		s.claimFactory.Metrics().SubchunkError()
		s.log.Error("decode subchunk entry", "error", err)
		return entry
	}
	s.claimFactory.Metrics().SubchunkDecoded()
	if int(index) >= len(virtualChunk.Sub()) {
		s.claimFactory.Metrics().SubchunkError()
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
			matched, ambiguous := singleClaimAt(claims, position.X(), position.Z())
			if ambiguous {
				s.claimFactory.Metrics().Reason(claim.QueryOverlap)
			}
			denied := !ambiguous && matched != nil &&
				!ClaimActionPermitted(*matched, actor, ClaimActionRender, position)
			s.claimFactory.Metrics().Action(uint8(ClaimActionRender), !denied)
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
	s.claimFactory.Metrics().SubchunkModified()
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
