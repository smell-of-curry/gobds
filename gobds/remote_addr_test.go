package gobds

import (
	"net"
	"testing"

	"github.com/df-mc/go-nethernet"
	"github.com/pion/webrtc/v4"
)

func TestRemoteAddrPort(t *testing.T) {
	udp := &net.UDPAddr{IP: net.ParseIP("203.0.113.9"), Port: 19132}
	if got, err := remoteAddrPort(udp); err != nil || got.String() != "203.0.113.9:19132" {
		t.Fatalf("raknet: got %v, %v", got, err)
	}

	nn := &nethernet.Addr{
		NetworkID:    "1",
		ConnectionID: 2,
		Candidates: []webrtc.ICECandidate{
			{Address: "10.0.2.15", Port: 36830, Typ: webrtc.ICECandidateTypeHost},
		},
		SelectedCandidate: &webrtc.ICECandidate{Address: "99.17.30.149", Port: 52948, Typ: webrtc.ICECandidateTypePrflx},
	}
	if got, err := remoteAddrPort(nn); err != nil || got.String() != "99.17.30.149:52948" {
		t.Fatalf("nethernet selected: got %v, %v", got, err)
	}
	nn.SelectedCandidate = nil
	if got, err := remoteAddrPort(nn); err != nil || got.String() != "10.0.2.15:36830" {
		t.Fatalf("nethernet fallback: got %v, %v", got, err)
	}
	nn.Candidates = nil
	if _, err := remoteAddrPort(nn); err == nil {
		t.Fatal("nethernet without candidates should error")
	}
}
