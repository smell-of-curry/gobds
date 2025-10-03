package session

import "github.com/df-mc/dragonfly/server/event"

// Context ...
type Context = event.Context[Conn]
