package authentication

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
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
			return nil, fmt.Errorf("service closed")
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

		switch response.StatusCode {
		case http.StatusNotFound:
			_ = response.Body.Close()
			return nil, ErrRecordNotFound
		case http.StatusGone:
			_ = response.Body.Close()
			return nil, fmt.Errorf("expired authentication record for: %s", xuid)
		case http.StatusOK:
			var responseModel ResponseModel
			if err = json.NewDecoder(response.Body).Decode(&responseModel); err != nil {
				_ = response.Body.Close()
				return nil, fmt.Errorf("failed to decode response: %w", err)
			}
			_ = response.Body.Close()
			return &responseModel, nil
		case http.StatusTooManyRequests:
			_ = response.Body.Close()
			lastErr = fmt.Errorf("rate limited")
			continue
		default:
			_ = response.Body.Close()
			lastErr = fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("authentication service unavailable")
	}
	return nil, lastErr
}

const mtuHintTimeout = 80 * time.Millisecond

// SendMTUOfIP asks the hub for a send-path MTU proven on this IP during
// the recent hub session. Returns 0 on miss/timeout so the caller keeps
// the conservative InitialSendMTU. Handshake-path — no retries.
func (s *Service) SendMTUOfIP(ip string) uint16 {
	if !s.Enabled || s.Closed || ip == "" {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), mtuHintTimeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/mtu?ip=%s", s.URL, url.QueryEscape(ip)), nil)
	if err != nil {
		return 0
	}
	request.Header.Set("authorization", s.Key)

	response, err := s.Client.Do(request)
	if err != nil {
		return 0
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0
	}
	var hint mtuHintModel
	if err = json.NewDecoder(response.Body).Decode(&hint); err != nil {
		return 0
	}
	if hint.SendMTU != 0 {
		s.Log.Info("applied hub send mtu hint", "ip", ip, "send_mtu", hint.SendMTU)
	}
	return hint.SendMTU
}
