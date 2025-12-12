package session

import (
	"sync/atomic"
	"time"
)

// Data ...
type Data struct {
	dimension atomic.Value
	gamemode  atomic.Value
	lastDrop  atomic.Value
	operator  atomic.Bool
}

// NewData ...
func NewData(conn Conn) *Data {
	gameData := conn.GameData()
	d := &Data{}
	d.dimension.Store(gameData.Dimension)
	d.gamemode.Store(gameData.PlayerGameMode)
	d.lastDrop.Store(time.Time{})
	d.operator.Store(false)
	return d
}

// Dimension ...
func (d *Data) Dimension() int32 {
	return d.dimension.Load().(int32)
}

// SetDimension ...
func (d *Data) SetDimension(dimension int32) {
	d.dimension.Store(dimension)
}

// GameMode ...
func (d *Data) GameMode() int32 {
	return d.gamemode.Load().(int32)
}

// SetGameMode ...
func (d *Data) SetGameMode(mode int32) {
	d.gamemode.Store(mode)
}

// SetLastDrop ...
func (d *Data) SetLastDrop() {
	d.lastDrop.Store(time.Now())
}

// InteractWithBlock ...
func (d *Data) InteractWithBlock() bool {
	lastDrop := d.lastDrop.Load().(time.Time)
	return time.Since(lastDrop) > time.Millisecond*500
}

// Operator ...
func (d *Data) Operator() bool {
	return d.operator.Load()
}

// SetOperator ...
func (d *Data) SetOperator(operator bool) {
	d.operator.Store(operator)
}
