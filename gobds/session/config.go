// Package session provides session management and packet handling for the GoBDS proxy.
package session

import (
	"log/slog"
	"time"

	"github.com/smell-of-curry/gobds/gobds/claim"
	"github.com/smell-of-curry/gobds/gobds/entity"
	"github.com/smell-of-curry/gobds/gobds/infra"
	"github.com/smell-of-curry/gobds/gobds/util/area"
)

// Config ...
type Config struct {
	Client Conn
	Server Conn

	AFKTimer *infra.AFKTimer
	Border   *area.Area2D

	ClaimPrefilter     bool
	ClaimDenyRendering bool
	Traffic            TrafficConfig
	TrafficMetrics     *TrafficMetrics

	EntityFactory *entity.Factory
	ClaimFactory  *claim.Factory

	Log *slog.Logger
}

// New ...
func (c Config) New() *Session {
	s := &Session{
		client: c.Client,
		server: c.Server,

		afkTimer: c.AFKTimer,
		border:   c.Border,

		claimPrefilter:     c.ClaimPrefilter,
		claimDenyRendering: c.ClaimDenyRendering,

		entityFactory: c.EntityFactory,
		claimFactory:  c.ClaimFactory,

		close: make(chan struct{}),
		corrective: correctiveState{
			last: make(map[correctiveKey]time.Time),
		},
		traffic: newTrafficState(c.Traffic, c.TrafficMetrics),

		lastForwardedPing: -1,

		data: NewData(c.Client),
		log:  c.Log,
	}
	s.afk.lastMoveTime = time.Now()
	s.afk.lastPosition = c.Client.GameData().PlayerPosition
	s.registerHandlers()
	return s
}
