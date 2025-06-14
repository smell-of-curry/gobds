package session

import (
	"sync/atomic"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
)

// Data ...
type Data struct {
	dimension atomic.Int32
	lastDrop  atomic.Pointer[time.Time]
}

// NewData ...
func NewData(client *minecraft.Conn) *Data {
	gameData := client.GameData()

	d := &Data{}
	d.dimension.Store(gameData.Dimension)
	d.lastDrop.Store(&time.Time{})
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

// SetLastDrop ...
func (d *Data) SetLastDrop() {
	now := time.Now()
	d.lastDrop.Store(&now)
}

// InteractWithContainer ...
func (d *Data) InteractWithContainer() bool {
	lastDrop := d.lastDrop.Load()
	return time.Since(*lastDrop) > time.Millisecond*500
}
