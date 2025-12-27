package authentication

import (
	"context"
	"errors"
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

var (
	// ErrRecordNotFound is returned when no authentication record is found.
	ErrRecordNotFound = errors.New("no authentication record found")
)

// AuthenticationOf ...
func (s *Service) AuthenticationOf(xuid string, ctx context.Context) (*ResponseModel, error) {
	if !s.Enabled {
		return &ResponseModel{Allowed: true}, nil
	}
	var lastErr error
	for attempt := 0; attempt <= service.MaxRetries; attempt++ {
		if s.Closed {
			break
		}
		if attempt > 0 {
			time.Sleep(service.RetryDelay)
		}

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s", s.URL, xuid), nil)
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
		case http.StatusNotFound:
			lastErr = ErrRecordNotFound
		case http.StatusGone:
			lastErr = fmt.Errorf("found expired authentication record found for: %s", xuid)
		case http.StatusOK:
			var responseModel ResponseModel
			if err = json.NewDecoder(response.Body).Decode(&responseModel); err != nil {
				return nil, err
			}
			return &responseModel, nil
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
