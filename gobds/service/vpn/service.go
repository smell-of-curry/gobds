package vpn

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4/json"
	"github.com/smell-of-curry/gobds/gobds/service"
)

// Service ...
type Service struct {
	*service.Service
	rateLimitReset time.Time
	mu             sync.Mutex
}

// NewService ...
func NewService(log *slog.Logger, c service.Config) *Service {
	return &Service{Service: service.NewService(log, c)}
}

// CheckIP ...
func (s *Service) CheckIP(ip string) (*ResponseModel, error) {
	if !s.Enabled {
		return &ResponseModel{Status: "success", Proxy: false}, nil
	}
	s.mu.Lock()
	if time.Now().Before(s.rateLimitReset) {
		s.mu.Unlock()
		return nil, fmt.Errorf("rate limit active, please wait until %v", s.rateLimitReset)
	}
	s.mu.Unlock()

	var lastErr error
	for attempt := 0; attempt <= 1; attempt++ {
		if s.Closed {
			break
		}
		if attempt > 0 {
			time.Sleep(service.RetryDelay)
		}

		url := fmt.Sprintf("%s/%s?fields=status,message,proxy", s.Url, ip)
		ctx, cancel := context.WithTimeout(context.Background(), service.RequestTimeout)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		response, err := s.Client.Do(request)
		cancel()
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			if service.ErrorIsTemporary(err) {
				continue
			}
			return nil, lastErr
		}

		s.handleRateLimitHeaders(response.Header)

		switch response.StatusCode {
		case http.StatusOK:
			var responseModel ResponseModel
			if err = json.NewDecoder(response.Body).Decode(&responseModel); err != nil {
				response.Body.Close()
				return nil, fmt.Errorf("failed to decode response body: %w", err)
			}
			response.Body.Close()
			if strings.EqualFold(responseModel.Status, "fail") {
				failMessage := responseModel.Message
				if strings.EqualFold(failMessage, "reserved range") {
					responseModel.Proxy = false
					return &responseModel, nil
				}
				return nil, fmt.Errorf("query failed: %s", failMessage)
			}
			return &responseModel, nil
		case http.StatusTooManyRequests:
			lastErr = fmt.Errorf("rate limited by api")
			time.Sleep(time.Duration(attempt+1) * service.RetryDelay)
			continue
		default:
			lastErr = fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
		response.Body.Close()
	}
	return nil, lastErr
}

// handleRateLimitHeaders ...
func (s *Service) handleRateLimitHeaders(header http.Header) {
	requestsRemainingStr := header.Get("X-Rl")
	timeToResetStr := header.Get("X-Ttl")

	if requestsRemainingStr == "0" && timeToResetStr != "" {
		ttl, err := strconv.Atoi(timeToResetStr)
		if err != nil {
			// couldn't parse header for whatever reason, just default to fallback wait time.
			ttl = 60
		}

		s.mu.Lock()
		s.rateLimitReset = time.Now().Add(time.Duration(ttl) * time.Second)
		s.mu.Unlock()
		s.Log.Warn("rate limit reached. waiting for reset.", "ttl_seconds", ttl)
	}
}
