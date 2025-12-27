package claim

import (
	"context"
	"fmt"
	"io"
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
			break
		}
		if attempt > 0 {
			time.Sleep(service.RetryDelay)
		}

		ctx, cancel := context.WithTimeout(context.Background(), service.RequestTimeout)
		defer cancel()
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		request.Header.Set("authorization", s.Key)

		response, err := s.Client.Do(request)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if service.ErrorIsTemporary(err) {
				continue
			}
			return nil, lastErr
		}
		defer func() { _ = response.Body.Close() }()

		switch response.StatusCode {
		case http.StatusOK:
			body, err := io.ReadAll(response.Body)
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				continue
			}

			var claimResponse []ResponseModel
			err = json.Unmarshal(body, &claimResponse)
			if err != nil {
				lastErr = fmt.Errorf("failed to unmarshal response body: %w", err)
				continue
			}

			s.Log.Info("fetched claims", "count", len(claimResponse))

			obj := make(map[string]PlayerClaim)
			for _, v := range claimResponse {
				obj[v.Key] = v.Data
			}
			return obj, nil
		case http.StatusTooManyRequests:
			lastErr = fmt.Errorf("rate limited")
			time.Sleep(time.Duration(attempt+1) * service.RetryDelay)
			continue
		default:
			lastErr = fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
	}
	return nil, lastErr
}
