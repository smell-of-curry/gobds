package session

import (
	"context"
	"net"

	"github.com/df-mc/dragonfly/server/session"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

// Conn ...
type Conn interface {
	session.Conn
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	GameData() minecraft.GameData
	StartGame(minecraft.GameData) error
	DoSpawn() error
	DoSpawnContext(ctx context.Context) error
	IdentityData() login.IdentityData
	ClientData() login.ClientData
}
