package session

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/df-mc/dragonfly/server/event"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/identity"
)

type identityTestConn struct {
	Conn
	identity login.IdentityData
	game     minecraft.GameData
}

func (c *identityTestConn) IdentityData() login.IdentityData { return c.identity }
func (c *identityTestConn) GameData() minecraft.GameData     { return c.game }

func TestScriptIdentityUsesAuthenticatedClientAndConsumesChallenge(t *testing.T) {
	received := make(chan identity.Assertion, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var a identity.Assertion
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			t.Error(err)
		}
		received <- a
		w.WriteHeader(204)
	}))
	defer server.Close()
	publisher, _ := identity.New("ONE", identity.Config{URL: server.URL, Key: "publisher"})
	client := &identityTestConn{identity: login.IdentityData{DisplayName: "Player", XUID: "1234"}}
	backend := &identityTestConn{identity: login.IdentityData{DisplayName: "Forged", XUID: "9999"}, game: minecraft.GameData{EntityUniqueID: -123}}
	s := &Session{client: client, server: backend, identityPublisher: publisher, close: make(chan struct{})}
	defer close(s.close)
	challenge := identity.Challenge{Version: 1, Instance: "ONE", BootID: "boot", SessionID: "join1", EntityID: "-123", Name: "Player", Nonce: strings.Repeat("a", 64), ExpiresAt: time.Now().Add(10 * time.Second).UnixMilli()}
	data, _ := json.Marshal(challenge)
	handler := &ScriptMessageHandler{}
	pkt := &packet.ScriptMessage{Identifier: identity.MessageID, Data: data}
	// A client cannot publish even a byte-for-byte copy of a valid challenge.
	clientContext := event.C[Conn](client)
	if err := handler.Handle(s, pkt, clientContext); err != nil || !clientContext.Cancelled() {
		t.Fatal("client identity message not consumed")
	}
	select {
	case <-received:
		t.Fatal("client spoof published")
	default:
	}
	serverContext := event.C[Conn](backend)
	if err := handler.Handle(s, pkt, serverContext); err != nil || !serverContext.Cancelled() {
		t.Fatal("backend challenge leaked")
	}
	select {
	case a := <-received:
		if a.XUID != "1234" || a.Name != "Player" {
			t.Fatal("publisher trusted upstream identity")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("assertion not delivered")
	}
	ordinary := event.C[Conn](backend)
	_ = handler.Handle(s, &packet.ScriptMessage{Identifier: "another:addon"}, ordinary)
	if ordinary.Cancelled() {
		t.Fatal("unrelated script message consumed")
	}
}
