package infra

import (
	"time"
)

// PingIndicator represents session ping indicator.
type PingIndicator struct {
	Identifier string
}

// AFKTimer represents session afk timer.
type AFKTimer struct {
	TimeoutDuration time.Duration
}
