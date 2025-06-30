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
