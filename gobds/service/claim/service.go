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
}

// NewService ...
func NewService(log *slog.Logger, c service.Config) *Service {
	return &Service{Service: service.NewService(log, c)}
}

// FetchClaims ...
func (s *Service) FetchClaims() (map[string]PlayerClaim, error) {
	if !s.Enabled {
		return map[string]PlayerClaim{}, nil
	}
	var lastErr error
	for attempt := 0; attempt <= service.MaxRetries; attempt++ {
		if s.Closed {
			return nil, fmt.Errorf("service closed")
		}
		if attempt > 0 {
			time.Sleep(service.RetryDelay)
		}

		ctx, cancel := context.WithTimeout(context.Background(), service.RequestTimeout)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		request.Header.Set("authorization", s.Key)

		response, err := s.Client.Do(request)
		if err != nil {
			cancel()
			lastErr = fmt.Errorf("request failed: %w", err)
			if service.ErrorIsTemporary(err) {
				continue
			}
			return nil, lastErr
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

			s.Log.Info("fetched claims", "count", len(claimResponse))

			obj := make(map[string]PlayerClaim, len(claimResponse))
			for _, v := range claimResponse {
				obj[v.Key] = v.Data
			}
			return obj, nil
		case http.StatusTooManyRequests:
			_ = response.Body.Close()
			cancel()
			lastErr = fmt.Errorf("rate limited")
			time.Sleep(time.Duration(attempt+1) * service.RetryDelay)
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
	return nil, lastErr
}
