package claim

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-jose/go-jose/v4/json"
	"github.com/smell-of-curry/gobds/gobds/service"
)

// Service ...
type Service struct {
	*service.Service
	lastModified string
}

// NewService ...
func NewService(c service.Config, log *slog.Logger) *Service {
	return &Service{Service: service.NewService(log, c)}
}

// FetchResult contains claim rows plus HTTP revalidation state.
type FetchResult struct {
	Claims       map[string]PlayerClaim
	NotModified  bool
	LastModified string
}

// FetchClaims ...
func (s *Service) FetchClaims() (FetchResult, error) {
	if !s.Enabled {
		return FetchResult{Claims: map[string]PlayerClaim{}}, nil
	}
	var lastErr error
	for attempt := 0; attempt <= service.MaxRetries; attempt++ {
		if s.Closed {
			return FetchResult{}, fmt.Errorf("service closed")
		}
		if attempt > 0 {
			time.Sleep(service.RetryDelay)
		}

		ctx, cancel := context.WithTimeout(context.Background(), service.RequestTimeout)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
		if err != nil {
			cancel()
			return FetchResult{}, fmt.Errorf("failed to create request: %w", err)
		}
		request.Header.Set("authorization", s.Key)
		if s.lastModified != "" {
			request.Header.Set("if-modified-since", s.lastModified)
		}

		response, err := s.Client.Do(request)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("request failed: %w", err)
			if service.ErrorIsTemporary(err) {
				continue
			}
			return FetchResult{}, lastErr
		}

		switch response.StatusCode {
		case http.StatusOK:
			var claimResponse []ResponseModel
			if err = json.NewDecoder(response.Body).Decode(&claimResponse); err != nil {
				_ = response.Body.Close()
				cancel()
				lastErr = fmt.Errorf("failed to decode response: %w", err)
				continue
			}
			_ = response.Body.Close()
			cancel()

			obj := make(map[string]PlayerClaim, len(claimResponse))
			for _, v := range claimResponse {
				if _, exists := obj[v.Key]; exists {
					return FetchResult{}, fmt.Errorf("duplicate claim response key %q", v.Key)
				}
				obj[v.Key] = v.Data
			}
			s.lastModified = response.Header.Get("last-modified")
			return FetchResult{
				Claims:       obj,
				LastModified: s.lastModified,
			}, nil
		case http.StatusNotModified:
			if modified := response.Header.Get("last-modified"); modified != "" {
				s.lastModified = modified
			}
			_ = response.Body.Close()
			cancel()
			return FetchResult{NotModified: true, LastModified: s.lastModified}, nil
		case http.StatusTooManyRequests:
			_ = response.Body.Close()
			cancel()
			lastErr = fmt.Errorf("rate limited")
			continue
		default:
			_ = response.Body.Close()
			cancel()
			lastErr = fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("claims service unavailable")
	}
	return FetchResult{}, lastErr
}
