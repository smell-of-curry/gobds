package service

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Service ...
type Service struct {
	Enabled bool
	URL     string
	Key     string
	Closed  bool

	Client *http.Client
	Log    *slog.Logger
}

const (
	// MaxRetries is the maximum number of retry attempts for service requests.
	MaxRetries = 3
	// RetryDelay is the delay between retry attempts.
	RetryDelay = 300 * time.Millisecond
	// RequestTimeout is the timeout duration for HTTP requests.
	RequestTimeout = 5 * time.Second
	// MaxConcurrentRequests is the maximum number of concurrent HTTP requests allowed.
	MaxConcurrentRequests = 5
)

// NewService ...
func NewService(log *slog.Logger, c Config) *Service {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   MaxConcurrentRequests * 2,
		MaxConnsPerHost:       MaxConcurrentRequests * 3,
	}

	return &Service{
		Enabled: c.Enabled,
		URL:     c.URL,
		Key:     c.Key,
		Closed:  false,
		Client: &http.Client{
			Timeout:   RequestTimeout,
			Transport: transport,
		},
		Log: log,
	}
}

// ErrorIsTemporary ...
func ErrorIsTemporary(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}
