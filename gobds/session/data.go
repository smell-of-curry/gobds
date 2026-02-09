package session

import (
	"sync/atomic"
	"time"

	"github.com/df-mc/dragonfly/server/player/skin"
)

// Data ...
type Data struct {
	dimension      atomic.Int32
	lastDrop       atomic.Pointer[time.Time]
	skin           atomic.Value
	lastSkinChange atomic.Value
}

// NewData ...
func NewData(conn Conn) *Data {
	gameData := conn.GameData()

	d := &Data{}
	d.dimension.Store(gameData.Dimension)
	d.lastDrop.Store(&time.Time{})
	d.skin.Store(skin.Skin{})
	d.lastSkinChange.Store(time.Time{})
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

// InteractWithBlock ...
func (d *Data) InteractWithBlock() bool {
	lastDrop := d.lastDrop.Load()
	return time.Since(*lastDrop) > time.Millisecond*500
}

// Skin ...
func (d *Data) Skin() skin.Skin {
	return d.skin.Load().(skin.Skin)
}

// SetSkin ...
func (d *Data) SetSkin(s skin.Skin) {
	d.skin.Store(s)
	d.lastSkinChange.Store(time.Now())
}

// LastSkinChange ...
func (d *Data) LastSkinChange() time.Time {
	return d.lastSkinChange.Load().(time.Time)
}

// CanChangeSkin ...
func (d *Data) CanChangeSkin(cooldown time.Duration) bool {
	t := d.lastSkinChange.Load().(time.Time)
	if t.IsZero() {
		return true
	}
	return time.Since(t) >= cooldown
}
