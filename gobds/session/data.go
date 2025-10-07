package session

import (
	"sync/atomic"
	"time"
)

// Data ...
type Data struct {
	dimension atomic.Value
	lastDrop  atomic.Value
}

// NewData ...
func NewData(conn Conn) *Data {
	d := &Data{}
	d.dimension.Store(conn.GameData().Dimension)
	d.lastDrop.Store(time.Time{})
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

// SetLastDrop ...
func (d *Data) SetLastDrop() {
	d.lastDrop.Store(time.Now())
}

// InteractWithBlock ...
func (d *Data) InteractWithBlock() bool {
	lastDrop := d.lastDrop.Load().(time.Time)
	return time.Since(lastDrop) > time.Millisecond*500
}
