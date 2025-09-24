package gobds

import (
	"io"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/smell-of-curry/gobds/gobds/session"
)

// Listener ...
type Listener interface {
	Accept() (session.Conn, error)
	Disconnect(conn session.Conn, reason string) error
	io.Closer
}

// listener ...
type listener struct {
	*minecraft.Listener
}

// Accept ...
func (l listener) Accept() (session.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return c.(session.Conn), nil
}

// Disconnect ...
func (l listener) Disconnect(conn session.Conn, reason string) error {
	return l.Listener.Disconnect(conn.(*minecraft.Conn), reason)
}
