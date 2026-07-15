package session

import (
	"sync/atomic"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Data ...
type Data struct {
	dimension atomic.Int32
	gamemode  atomic.Int32
	lastDrop  atomic.Pointer[time.Time]
	operator  atomic.Bool
}

// NewData ...
func NewData(conn Conn) *Data {
	gameData := conn.GameData()
	d := &Data{}
	d.dimension.Store(gameData.Dimension)
	d.gamemode.Store(gameData.PlayerGameMode)
	d.lastDrop.Store(&time.Time{})
	d.operator.Store(gameData.PlayerPermissions == packet.PermissionLevelOperator)
	return d
}

// Dimension ...
func (d *Data) Dimension() int32 {
	return d.dimension.Load()
}

// SetDimension ...
func (d *Data) SetDimension(dimension int32) {
	d.dimension.Store(dimension)
}

// GameMode ...
func (d *Data) GameMode() int32 {
	return d.gamemode.Load()
}

// SetGameMode ...
func (d *Data) SetGameMode(mode int32) {
	d.gamemode.Store(mode)
}

// SetLastDrop ...
func (d *Data) SetLastDrop() {
	now := time.Now()
	d.lastDrop.Store(&now)
}

// InteractWithBlock ...
func (d *Data) InteractWithBlock() bool {
	return time.Since(*d.lastDrop.Load()) > time.Millisecond*500
}

// Operator ...
func (d *Data) Operator() bool {
	return d.operator.Load()
}

// SetOperator ...
func (d *Data) SetOperator(operator bool) {
	d.operator.Store(operator)
}
