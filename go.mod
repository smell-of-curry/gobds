module github.com/smell-of-curry/gobds

go 1.24.1

require (
	github.com/avast/retry-go/v4 v4.6.1
	github.com/df-mc/dragonfly v0.10.4-0.20250525055426-6759f0a7617b
	github.com/getsentry/sentry-go v0.34.0
	github.com/go-gl/mathgl v1.2.0
	github.com/go-jose/go-jose/v4 v4.1.0
	github.com/gofrs/flock v0.12.1
	github.com/google/uuid v1.6.0
	github.com/restartfu/gophig v0.0.2
	github.com/sandertv/go-raknet v1.14.3-0.20250525005230-991ee492a907
	github.com/sandertv/gophertunnel v1.46.0
	github.com/tailscale/hujson v0.0.0-20250226034555-ec1d1c113d33
)

require (
	github.com/brentp/intintmap v0.0.0-20190211203843-30dc0ade9af9 // indirect
	github.com/df-mc/goleveldb v1.1.9 // indirect
	github.com/df-mc/worldupgrader v1.0.19 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/muhammadmuzzammil1998/jsonc v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/rogpeppe/go-internal v1.9.0 // indirect
	github.com/segmentio/fasthash v1.0.3 // indirect
	golang.org/x/exp v0.0.0-20250606033433-dcc06ee1d476 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/sandertv/gophertunnel => github.com/smell-of-curry/gophertunnel v1.46.1-0.20250704012025-8e86404b4050

replace github.com/sandertv/go-raknet => github.com/smell-of-curry/go-raknet v0.0.0-20250525005230-991ee492a907
