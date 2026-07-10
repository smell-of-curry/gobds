package vpn

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
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

	// whitelist holds CIDR ranges never treated as proxies, regardless of
	// what the upstream detection API says.
	whitelist []netip.Prefix
}

// NewService creates a VPN detection service. whitelistCIDRs are IP ranges
// (e.g. "45.230.64.0/22") that always pass the check; invalid entries are
// logged and skipped.
func NewService(log *slog.Logger, c service.Config, whitelistCIDRs []string) *Service {
	whitelist := make([]netip.Prefix, 0, len(whitelistCIDRs))
	for _, cidr := range whitelistCIDRs {
		p, err := netip.ParsePrefix(strings.TrimSpace(cidr))
		if err != nil {
			log.Warn("ignoring invalid vpn whitelist cidr", "cidr", cidr, "error", err)
			continue
		}
		whitelist = append(whitelist, p.Masked())
	}
	return &Service{Service: service.NewService(log, c), whitelist: whitelist}
}

// CheckIP ...
func (s *Service) CheckIP(ip string, ctx context.Context) (*ResponseModel, error) {
	if !s.Enabled {
		return &ResponseModel{Status: "success", Proxy: false}, nil
	}
	if s.isWhitelisted(ip) {
		return &ResponseModel{Status: "success", Proxy: false}, nil
	}
	if active, reset := s.rateLimitActive(); active {
		return nil, fmt.Errorf("rate limit active, please wait until %v", reset)
	}

	var lastErr error
	for attempt := 0; attempt <= 1; attempt++ {
		if s.Closed {
			return nil, fmt.Errorf("service closed")
		}
		if attempt > 0 {
			time.Sleep(service.RetryDelay)
		}

		url := fmt.Sprintf("%s/%s?fields=status,message,proxy", s.URL, ip)
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		response, err := s.Client.Do(request)
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
				_ = response.Body.Close()
				return nil, fmt.Errorf("failed to decode response body: %w", err)
			}
			_ = response.Body.Close()
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
			_ = response.Body.Close()
			lastErr = fmt.Errorf("rate limited by api")
			continue
		default:
			_ = response.Body.Close()
			lastErr = fmt.Errorf("unexpected status code: %d", response.StatusCode)
		}
	}
	return nil, lastErr
}

func (s *Service) isWhitelisted(ip string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	for _, p := range s.whitelist {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}

func (s *Service) rateLimitActive() (bool, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return time.Now().Before(s.rateLimitReset), s.rateLimitReset
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
