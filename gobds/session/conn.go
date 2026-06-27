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

packets: make(chan *packetData, 8),   // nur 8 Slots Puffer!

func (conn *Conn) receive(data []byte) error {
    ...
    if conn.loggedIn && !conn.waitingForSpawn.Load() {
        select {
        case previous := <-conn.packets:
            // Channel ist voll → ältestes Packet raus, in deferredPackets verschieben
            conn.deferPacket(previous)
        default:
        }
        select {
        case conn.packets <- pkData:   // neues Packet rein
        }
        return nil
    }
    ...
}