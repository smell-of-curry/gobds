package gobds

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// gophertunnel defers the upstream registry until ReadPacket, but its downstream
// StartGame sends an empty registry. Substitute this player's BDS packet at that
// exact point, before the block palette is loaded.
type voxelProtocol struct {
	minecraft.Protocol
	registries sync.Map // *minecraft.Conn -> *packet.VoxelShapes, only during login
}

func (p *voxelProtocol) ConvertFromLatest(pk packet.Packet, conn *minecraft.Conn) []packet.Packet {
	if _, ok := pk.(*packet.VoxelShapes); ok {
		if shapes, loaded := p.registries.LoadAndDelete(conn); loaded {
			return []packet.Packet{shapes.(*packet.VoxelShapes)}
		}
	}
	return p.Protocol.ConvertFromLatest(pk, conn)
}

type bufferedUpstream struct {
	session.Conn
	pending []packet.Packet
}

func (c *bufferedUpstream) ReadPacket() (packet.Packet, error) {
	if len(c.pending) == 0 {
		return c.Conn.ReadPacket()
	}
	pk := c.pending[0]
	c.pending[0] = nil
	c.pending = c.pending[1:]
	return pk, nil
}

// Dial has already received StartGame and ItemRegistry. All preceding packets
// are in its deferred queue. Preserve their order for the normal session handlers.
func prepareVoxelRegistry(client, backend session.Conn) (session.Conn, func(), error) {
	noCleanup := func() {}
	down, ok := client.(*minecraft.Conn)
	if !ok {
		return backend, noCleanup, nil
	}
	p, ok := down.Proto().(*voxelProtocol)
	up, hasDeadline := backend.(*minecraft.Conn)
	if !ok || !hasDeadline {
		return backend, noCleanup, nil
	}
	// A backend that omitted the packet must still be able to finish spawning.
	if err := up.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		return nil, noCleanup, err
	}
	defer func() { _ = up.SetReadDeadline(time.Time{}) }()
	buffered := &bufferedUpstream{Conn: backend}
	// Preserve normal forwarding for non-BDS servers with long preambles.
	for len(buffered.pending) < 256 {
		pk, err := up.ReadPacket()
		if errors.Is(err, context.DeadlineExceeded) {
			return buffered, noCleanup, nil
		}
		if err != nil {
			return nil, noCleanup, fmt.Errorf("read upstream voxel registry: %w", err)
		}
		if shapes, ok := pk.(*packet.VoxelShapes); ok {
			p.registries.Store(down, shapes)
			return buffered, func() { p.registries.Delete(down) }, nil
		}
		buffered.pending = append(buffered.pending, pk)
	}
	return buffered, noCleanup, nil
}
