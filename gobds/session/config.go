// Package session provides session management and packet handling for the GoBDS proxy.
package session

import (
	"log/slog"

	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/util/area"
)

// Config ...
type Config struct {
	Client        Conn
	Server        Conn
	PingIndicator *infra.PingIndicator
	AfkTimer      *infra.AFKTimer
	Border        *area.Area2D
	Log           *slog.Logger
}

// New ...
func (c Config) New() *Session {
	s := &Session{
		client: c.Client,
		server: c.Server,

		pingIndicator: c.PingIndicator,
		afkTimer:      c.AfkTimer,
		border:        c.Border,

		close: make(chan struct{}),

		data: NewData(c.Client),
		log:  c.Log,
	}
	s.registerHandlers()
	return s
}
