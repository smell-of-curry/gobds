package infra

import (
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/service/vpn"
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
