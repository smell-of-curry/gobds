// Package identity publishes connection identities to a scoped manager broker.
package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const MessageID = "engine:identity_challenge"

type Config struct {
	URL string
	Key string
}
type Challenge struct {
	Version   int    `json:"version"`
	Instance  string `json:"instance"`
	BootID    string `json:"bootId"`
	SessionID string `json:"sessionId"`
	EntityID  string `json:"entityId"`
	Name      string `json:"name"`
	Nonce     string `json:"nonce"`
	ExpiresAt int64  `json:"expiresAt"`
}
type Assertion struct {
	Challenge
	XUID string `json:"xuid"`
}
type Publisher struct {
	instance string
	config   Config
	client   *http.Client
}

var decimal = regexp.MustCompile(`^[1-9][0-9]{0,19}$`)
var noncePattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var sessionPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,80}$`)

// New permits cleartext only over loopback. Redirects never receive credentials.
func New(instance string, config Config) (*Publisher, error) {
	if config.URL == "" && config.Key == "" {
		return nil, nil
	}
	u, err := url.Parse(config.URL)
	if err != nil || config.Key == "" || u.User != nil || u.Fragment != "" || u.Host == "" {
		return nil, errors.New("invalid identity endpoint")
	}
	ip := net.ParseIP(u.Hostname())
	if u.Scheme != "https" && !(u.Scheme == "http" && ip != nil && ip.IsLoopback()) {
		return nil, errors.New("identity endpoint requires TLS or loopback")
	}
	return &Publisher{instance: strings.ToUpper(instance), config: config, client: &http.Client{
		Timeout:       4 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("identity redirect refused") },
	}}, nil
}

// Decode binds the challenge to this authenticated client and its backend entity.
func (p *Publisher) Decode(data []byte, entityID, name, xuid string, now time.Time) (Assertion, bool) {
	if p == nil || len(data) > 2048 || !decimal.MatchString(xuid) {
		return Assertion{}, false
	}
	var c Challenge
	if json.Unmarshal(data, &c) != nil || c.Version != 1 || c.Instance != p.instance || c.EntityID != entityID || c.Name != name ||
		c.BootID == "" || len(c.BootID) > 80 || !sessionPattern.MatchString(c.SessionID) || !noncePattern.MatchString(c.Nonce) ||
		c.ExpiresAt <= now.UnixMilli() || c.ExpiresAt > now.Add(20*time.Second).UnixMilli() {
		return Assertion{}, false
	}
	return Assertion{Challenge: c, XUID: xuid}, true
}

// Publish sends only a validated assertion, never a command supplied by the addon.
func (p *Publisher) Publish(ctx context.Context, assertion Assertion) error {
	body, err := json.Marshal(assertion)
	if err != nil {
		return errors.New("identity encoding failed")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.config.URL, bytes.NewReader(body))
	if err != nil {
		return errors.New("identity request failed")
	}
	req.Header.Set("Authorization", p.config.Key)
	req.Header.Set("Content-Type", "application/json")
	res, err := p.client.Do(req)
	if err != nil {
		return errors.New("identity delivery failed")
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 2048))
	if res.StatusCode != http.StatusNoContent {
		return errors.New("identity assertion rejected")
	}
	return nil
}
