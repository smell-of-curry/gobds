package gobds

// Config ...
type Config struct {
	Network struct {
		LocalAddress  string
		RemoteAddress string

		Whitelisted bool

		SecuredSlots      int
		MaxRenderDistance int
		FlushRate         int
	}
	Border struct {
		Enabled    bool
		MinX, MinZ int32
		MaxX, MaxZ int32
	}
	Resources struct {
		PacksRequired bool

		URLResources  []string
		PathResources []string
	}
	Services struct {
		IdentityService struct {
			URL string
			Key string
		}
		ClaimService struct {
			URL string
			Key string
		}
	}
	Encryption struct {
		Key string
	}
}
