package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// LevelChunkHandler ...
type LevelChunkHandler struct{}

// Handle ...
func (*LevelChunkHandler) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.LevelChunk)

	if s.border == nil {
		return nil
	}

	if !s.border.ChunkInside(pkt.Position) {
		ctx.Cancel()
	}
	return nil
}
