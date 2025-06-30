package vpn

const (
	StatusSuccess = "success"
	StatusFail    = "fail"
)

// ResponseModel ...
type ResponseModel struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
	Proxy   bool   `json:"proxy"`
}
