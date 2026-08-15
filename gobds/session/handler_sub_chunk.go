package session

import (
	"bytes"
	"sync"
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

// pooled decode buffers — payload sizes vary; Grow retains capacity across uses.
var subChunkPayloadPool = sync.Pool{
	New: func() any { return new(bytes.Buffer) },
}

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
	snapshot, snapshotStatus, dimension, dimensionFound := resolveClaimSubChunkContext(s, pkt.Dimension, dimensions)
	dimensionRange, rangeFound := dimensionRangeByID(pkt.Dimension, dimensions)

	entries := make([]protocol.SubChunkEntry, 0, len(originalEntries))
	for _, entry := range originalEntries {
		kept, ok := filterSubChunkEntry(
			s, pkt, entry, snapshot, snapshotStatus,
			dimension, dimensionFound, dimensionRange, rangeFound,
		)
		if ok {
			entries = append(entries, kept)
		}
	}
	pkt.SubChunkEntries = entries
	return nil
}

func resolveClaimSubChunkContext(
	s *Session,
	dimensionID int32,
	dimensions []protocol.DimensionDefinition,
) (*claim.Snapshot, claim.QueryStatus, string, bool) {
	if !s.claimDenyRendering || s.claimFactory == nil {
		return nil, 0, "", false
	}
	snapshot, snapshotStatus := s.claimFactory.Snapshot(time.Now())
	dimension, dimensionFound := claimDimensionFromInt(dimensionID, dimensions)
	if snapshotStatus != claim.QueryReady {
		s.claimFactory.Metrics().Reason(snapshotStatus)
	}
	if !dimensionFound {
		s.claimFactory.Metrics().Reason(claim.QueryUnknownDimension)
	}
	return snapshot, snapshotStatus, dimension, dimensionFound
}

// filterSubChunkEntry returns the (possibly rewritten) entry and whether to keep it.
func filterSubChunkEntry(
	s *Session,
	pkt *packet.SubChunk,
	entry protocol.SubChunkEntry,
	snapshot *claim.Snapshot,
	snapshotStatus claim.QueryStatus,
	dimension string,
	dimensionFound bool,
	dimensionRange cube.Range,
	rangeFound bool,
) (protocol.SubChunkEntry, bool) {
	chunkPos := protocol.ChunkPos{
		pkt.Position.X() + int32(entry.Offset[0]),
		pkt.Position.Z() + int32(entry.Offset[2]),
	}

	if s.border != nil && !s.border.ChunkInside(chunkPos) {
		return protocol.SubChunkEntry{}, false
	}

	if !s.claimDenyRendering || s.claimFactory == nil {
		return entry, true
	}

	// GoBDS disables the backend blob cache, so successful entries normally
	// contain the sub-chunk bytes inline. Leave cached entries untouched if
	// that invariant changes.
	if entry.Result != protocol.SubChunkResultSuccess || !rangeFound || !dimensionFound ||
		snapshotStatus != claim.QueryReady || pkt.CacheEnabled {
		return entry, true
	}
	chunkX := float32(chunkPos.X() << 4)
	chunkZ := float32(chunkPos.Z() << 4)
	candidates := snapshot.Candidates(dimension, chunkX, chunkZ)
	s.claimFactory.Metrics().Candidates(len(candidates))

	return applyClaimDenyBlocks(
		s,
		pkt,
		entry,
		chunkPos,
		dimensionRange,
		candidates,
		denyBlockRuntimeID(s.GameData().UseBlockNetworkIDHashes),
		ClaimActor{XUID: s.IdentityData().XUID, Operator: s.Data().Operator()},
	), true
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

	var mask [16]uint16
	forwarded, denied := floorDenyMask(claims, actor, chunkPos.X()<<4, chunkPos.Z()<<4, &mask)
	s.claimFactory.Metrics().ActionN(uint8(ClaimActionRender), uint64(forwarded), uint64(denied))
	if denied == 0 {
		return entry
	}

	rawPayload, ok := entry.RawPayload.Value()
	if !ok {
		return entry
	}

	buf := subChunkPayloadPool.Get().(*bytes.Buffer)
	buf.Reset()
	buf.Write(rawPayload)
	defer func() {
		buf.Reset()
		subChunkPayloadPool.Put(buf)
	}()

	virtualChunk := chunk.New(world.DefaultBlockRegistry, dimensionRange)
	var index byte
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
	// Remaining bytes after the sub-chunk body are block-entity NBT; only clone
	// once we know we will rewrite the entry.
	blockEntityPayload := bytes.Clone(buf.Bytes())

	for z := uint8(0); z < 16; z++ {
		row := mask[z]
		if row == 0 {
			continue
		}
		for x := uint8(0); x < 16; x++ {
			if row&(1<<x) == 0 {
				continue
			}
			decodedEntry.SetBlock(x, 0, z, 0, denyID)
		}
	}
	s.claimFactory.Metrics().SubchunkModified()
	virtualChunk.Sub()[index] = decodedEntry
	entry.RawPayload = protocol.Option(append(
		chunk.EncodeSubChunk(virtualChunk, chunk.NetworkEncoding, int(index)),
		blockEntityPayload...,
	))
	return entry
}

// decodeSubChunk links Dragonfly's unexported single-subchunk decoder.
//
//go:linkname decodeSubChunk github.com/df-mc/dragonfly/server/world/chunk.decodeSubChunk
func decodeSubChunk(buf *bytes.Buffer, c *chunk.Chunk, index *byte, e chunk.Encoding) (*chunk.SubChunk, error)
