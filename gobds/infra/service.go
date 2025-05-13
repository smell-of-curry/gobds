package infra

import (
	"github.com/smell-of-curry/gobds/gobds/service/claim"
	"github.com/smell-of-curry/gobds/gobds/service/identity"
)

var (
	IdentityService *identity.Service
	ClaimService    *claim.Service
)
