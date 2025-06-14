package interceptor

import (
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Client ...
type Client interface {
	SendPingIndicator()
	Ping() int64

	Data() any

	WriteToClient(pk packet.Packet)
	WriteToServer(pk packet.Packet)

	GameData() minecraft.GameData
	ClientData() login.ClientData
	Locale() string
	IdentityData() login.IdentityData
}

// ClientData ...
type ClientData interface {
	Dimension() int32
	SetDimension(dimension int32)
	SetLastDrop()
	InteractWithContainer() bool
}
