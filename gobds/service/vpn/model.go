// Package vpn provides VPN detection service models and types.
package vpn

const (
	// StatusSuccess indicates a successful VPN check response.
	StatusSuccess = "success"
	// StatusFail indicates a failed VPN check response.
	StatusFail = "fail"
)

// ResponseModel ...
type ResponseModel struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Proxy   bool   `json:"proxy"`
}
