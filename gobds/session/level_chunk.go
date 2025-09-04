package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// LevelChunk ...
type LevelChunk struct{}

// Handle ...
func (*LevelChunk) Handle(s *Session, pk packet.Packet, ctx *Context) error {
	pkt := pk.(*packet.LevelChunk)

	if s.border == nil {
		return nil
	}

	if !s.border.ChunkInside(pkt.Position) {
		ctx.Cancel()
	}
	return nil
}
