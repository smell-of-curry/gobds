package interceptor

import (
	"sync"

	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// Interceptor ...
type Interceptor struct {
	handlers map[int]Handler
	mu       sync.RWMutex
}

// NewInterceptor ...
func NewInterceptor() *Interceptor {
	return &Interceptor{
		handlers: make(map[int]Handler),
	}
}

// Intercept ...
func (i *Interceptor) Intercept(c Client, pk packet.Packet, ctx *session.Context) {
	h, ok := i.HandlerOf(pk)
	if !ok {
		return
	}
	h.Handle(c, pk, ctx)
}

// AddHandler ...
func (i *Interceptor) AddHandler(id int, h Handler) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.handlers[id] = h
}

// HandlerOf ...
func (i *Interceptor) HandlerOf(pk packet.Packet) (Handler, bool) {
	i.mu.RLock()
	defer i.mu.RUnlock()
	h, ok := i.handlers[int(pk.ID())]
	return h, ok
}
