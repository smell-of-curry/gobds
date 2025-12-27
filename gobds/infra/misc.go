package infra

import (
	"fmt"
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

// SkinConfig represents configuration for session skin handling.
type SkinConfig struct {
	SkinChangeCooldown time.Duration
	HeadsDirectory     string
	HeadServiceURL     string
}

// HeadURL ...
func (c *SkinConfig) HeadURL(xuid string) string {
	return fmt.Sprintf("%s/heads/%s", c.HeadServiceURL, xuid)
}
