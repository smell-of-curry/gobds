// Package session provides session management and packet handling for the GoBDS proxy.
package session

import (
	"log/slog"

	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/util/area"
)

// Config ...
type Config struct {
	Client Conn
	Server Conn

	PingIndicator *infra.PingIndicator
	AFKTimer      *infra.AFKTimer
	Border        *area.Area2D

	EntityFactory *entity.Factory
	ClaimFactory  *claim.Factory

	Log *slog.Logger
}

// New ...
func (c Config) New() *Session {
	s := &Session{
		client: c.Client,
		server: c.Server,

		pingIndicator: c.PingIndicator,
		afkTimer:      c.AFKTimer,
		border:        c.Border,

		entityFactory: c.EntityFactory,
		claimFactory:  c.ClaimFactory,

		close: make(chan struct{}),

		data: NewData(c.Client),
		log:  c.Log,
	}
	s.registerHandlers()
	return s
}
