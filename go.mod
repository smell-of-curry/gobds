module github.com/smell-of-curry/gobds

go 1.24.1

require (
	github.com/go-jose/go-jose/v4 v4.1.0
	github.com/google/uuid v1.6.0
	github.com/sandertv/go-raknet v1.14.3-0.20250305181847-6af3e95113d6
	github.com/sandertv/gophertunnel v1.46.0
	github.com/tailscale/hujson v0.0.0-20250226034555-ec1d1c113d33
)

require (
	github.com/go-gl/mathgl v1.2.0 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/muhammadmuzzammil1998/jsonc v1.0.0 // indirect
	golang.org/x/net v0.40.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/text v0.25.0 // indirect
)

replace github.com/sandertv/gophertunnel => github.com/smell-of-curry/gophertunnel v1.46.1-0.20250509001105-c57985510556

replace github.com/sandertv/go-raknet => github.com/smell-of-curry/go-raknet v0.0.0-20250314172126-71bb156b6413
