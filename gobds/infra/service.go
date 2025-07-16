package infra

import (
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/service/vpn"
	"github.com/smell-of-curry/gobds/gobds/util"
)

var (
	AuthenticationService *authentication.Service
	ClaimService          *claim.Service
	VPNService            *vpn.Service
)

// PingIndicator represents the global config for the session ping indicator.
var PingIndicator struct {
	Enabled    bool
	Identifier string
}

// AFKTimer represents the global config for the session afk timer.
var AFKTimer struct {
	Enabled         bool
	TimeoutDuration util.Duration
}
