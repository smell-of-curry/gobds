// Package authentication provides authentication service models and types.
package authentication

// ResponseModel ...
type ResponseModel struct {
	Allowed bool   `json:"allowed"`
	SendMTU uint16 `json:"send_mtu,omitempty"`
}

// mtuHintModel is the hub /authentication/mtu/:ip response.
type mtuHintModel struct {
	SendMTU uint16 `json:"send_mtu"`
}
