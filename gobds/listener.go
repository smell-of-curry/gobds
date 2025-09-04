package gobds

import (
	"io"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/smell-of-curry/gobds/gobds/session"
)

type Listener interface {
	Accept() (session.Conn, error)
	Disconnect(conn session.Conn, reason string) error
	io.Closer
}

type DefaultListener struct {
	*minecraft.Listener
}

func (l DefaultListener) Accept() (session.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return c.(session.Conn), nil
}

func (l DefaultListener) Disconnect(conn session.Conn, reason string) error {
	return l.Listener.Disconnect(conn.(*minecraft.Conn), reason)
}
