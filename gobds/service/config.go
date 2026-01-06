// Package service provides HTTP-based service clients for authentication, VPN checks, and claims.
package service

// Config ...
type Config struct {
	Enabled bool
	URL     string
	Key     string
}
