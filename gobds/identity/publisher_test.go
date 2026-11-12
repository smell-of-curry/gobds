package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func fixture() Challenge {
	return Challenge{1, "ONE", "boot", "join1", "-123", "Player", strings.Repeat("a", 64), time.Now().Add(10 * time.Second).UnixMilli()}
}
func TestDecodeBindsEverySessionField(t *testing.T) {
	p, err := New("one", Config{"http://127.0.0.1:1234/identity/ONE/assert", "secret"})
	if err != nil {
		t.Fatal(err)
	}
	c := fixture()
	data, _ := json.Marshal(c)
	if a, ok := p.Decode(data, "-123", "Player", "1234", time.Now()); !ok || a.XUID != "1234" {
		t.Fatal("valid identity rejected")
	}
	for _, tc := range []struct{ name, entity, player, xuid string }{
		{"entity", "-124", "Player", "1234"}, {"name", "-123", "Other", "1234"}, {"emptyXUID", "-123", "Player", ""}, {"injection", "-123", "Player", "123\nstop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := p.Decode(data, tc.entity, tc.player, tc.xuid, time.Now()); ok {
				t.Fatal("invalid binding accepted")
			}
		})
	}
	for _, change := range []func(*Challenge){
		func(c *Challenge) { c.Instance = "TWO" }, func(c *Challenge) { c.Version = 2 }, func(c *Challenge) { c.Nonce = "short" }, func(c *Challenge) { c.ExpiresAt = 1 }, func(c *Challenge) { c.ExpiresAt = time.Now().Add(time.Hour).UnixMilli() }, func(c *Challenge) { c.SessionID = "" },
	} {
		invalid := c
		change(&invalid)
		b, _ := json.Marshal(invalid)
		if _, ok := p.Decode(b, "-123", "Player", "1234", time.Now()); ok {
			t.Fatal("invalid challenge accepted")
		}
	}
	if _, ok := p.Decode([]byte(strings.Repeat("x", 2049)), "-123", "Player", "1234", time.Now()); ok {
		t.Fatal("oversized challenge accepted")
	}
}

func TestPublishUsesScopedKeyAndExactPayload(t *testing.T) {
	expected := Assertion{fixture(), "1234"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "publisher-only" {
			t.Error("wrong request credentials/method")
		}
		var got Assertion
		if json.NewDecoder(r.Body).Decode(&got) != nil || got != expected {
			t.Error("wrong payload")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	p, err := New("ONE", Config{server.URL, "publisher-only"})
	if err != nil {
		t.Fatal(err)
	}
	if err = p.Publish(context.Background(), expected); err != nil {
		t.Fatal(err)
	}
}

func TestEndpointValidationAndRedirectRefusal(t *testing.T) {
	for _, u := range []string{"http://example.com/identity", "file:///tmp/key", "https://user:password@example.com/"} {
		if _, err := New("ONE", Config{u, "secret"}); err == nil {
			t.Fatalf("accepted %s", u)
		}
	}
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("redirect target contacted") }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	p, _ := New("ONE", Config{server.URL, "secret"})
	if err := p.Publish(context.Background(), Assertion{fixture(), "1234"}); err == nil {
		t.Fatal("redirect accepted")
	}
	if p, err := New("ONE", Config{}); p != nil || err != nil {
		t.Fatal("unconfigured publisher must stay disabled")
	}
}
