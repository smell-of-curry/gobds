package infra

import (
	"github.com/smell-of-curry/gobds/gobds/service/authentication"
	"github.com/smell-of-curry/gobds/gobds/service/claim"
)

var (
	AuthenticationService *authentication.Service
	ClaimService          *claim.Service
)
