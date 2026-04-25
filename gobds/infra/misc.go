// Package infra provides infrastructure level components & utilities.
package infra

import (
	"time"
)

// PingIndicator represents session ping indicator.
type PingIndicator struct {
	Identifier string
}

// AFKTimer represents session afk timer configuration.
//
// TimeoutDuration is the idle duration after which a player becomes eligible
// to be kicked for being AFK. The kick itself only fires when the server is at
// least FullnessThreshold full.
//
// WarnApproaching and MarkAFK drive always-on soft warnings regardless of
// server fullness. FinalWarning is only sent when the server is at or above
// FullnessThreshold.
type AFKTimer struct {
	TimeoutDuration   time.Duration
	WarnApproaching   time.Duration
	MarkAFK           time.Duration
	FinalWarning      time.Duration
	FullnessThreshold float64
}
