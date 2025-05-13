package session

import (
	"sync/atomic"

	"github.com/sandertv/gophertunnel/minecraft"
)

// Data ...
type Data struct {
	dimension atomic.Int32
}

// NewData ...
func NewData(client *minecraft.Conn) *Data {
	gameData := client.GameData()

	d := &Data{}
	d.dimension.Store(gameData.Dimension)
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
