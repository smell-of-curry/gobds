package gobds

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestXUIDReservationRejectsSimultaneousAndAllowsReconnect(t *testing.T) {
	server := &Server{}
	if !server.ReserveXUID("123") {
		t.Fatal("first reservation rejected")
	}
	if server.ReserveXUID("123") {
		t.Fatal("simultaneous reservation accepted")
	}
	server.ReleaseXUID("123")
	if !server.ReserveXUID("123") {
		t.Fatal("reconnect rejected after release")
	}
}

func TestXUIDReservationIsAtomic(t *testing.T) {
	server := &Server{}
	var accepted atomic.Int32
	var group sync.WaitGroup
	for range 32 {
		group.Add(1)
		go func() {
			defer group.Done()
			if server.ReserveXUID("same") {
				accepted.Add(1)
			}
		}()
	}
	group.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("accepted %d simultaneous reservations", accepted.Load())
	}
}
