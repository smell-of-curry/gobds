package handlers

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/interceptor"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// LevelChunk ...
type LevelChunk struct{}

// Handle ...
func (LevelChunk) Handle(_ interceptor.Client, pk packet.Packet, ctx *session.Context) {
	pkt := pk.(*packet.LevelChunk)

	if !infra.WorldBorderEnabled() {
		return
	}

	if !infra.WorldBorder.ArePositionsInside(pkt.Position[:]) {
		ctx.Cancel()
	}
}
