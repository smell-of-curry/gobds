package session

import (
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// AddPlayerHandler fixes BDS→client AddPlayer packets that would disconnect
// viewers on 1.26.40. BuildPlatform 0 is not a valid DeviceOS (starts at 1);
// clients reject it with disconnect reason "Block" (df-mc/dragonfly#1382).
type AddPlayerHandler struct{}

// Handle ...
func (*AddPlayerHandler) Handle(_ *Session, pk packet.Packet, _ *Context) error {
	pkt := pk.(*packet.AddPlayer)
	if pkt.BuildPlatform == 0 {
		pkt.BuildPlatform = int32(protocol.DeviceDedicated)
	}
	return nil
}
