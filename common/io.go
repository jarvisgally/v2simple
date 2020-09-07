package common

import (
	"bytes"
	"errors"
	"io"
	"net"
)

const (
	TypeHttp = iota
	TypeUnknown
)

var (
	httpMethods = [...][]byte{
		[]byte("GET"),
		[]byte("POST"),
		[]byte("HEAD"),
		[]byte("PUT"),
		[]byte("DELETE"),
		[]byte("OPTIONS"),
		[]byte("CONNECT"),
		[]byte("PRI"),
	}
	sep = []byte(" ")
)

type SniffConn struct {
	net.Conn
	rout         io.Reader
	peeked, read bool
	peeks        []byte
}

func NewSniffConn(c net.Conn) *SniffConn {
	s := &SniffConn{Conn: c, rout: c}
	return s
}

func (c *SniffConn) Read(p []byte) (int, error) {
	if !c.read {
		c.read = true
		c.rout = io.MultiReader(bytes.NewReader(c.peeks), c.Conn)
	}
	return c.rout.Read(p)
}

func (c *SniffConn) Sniff() int {
	var err error
	c.peeks, err = c.peek(64)
	if err != nil && err != io.EOF {
		return TypeUnknown
	}

	if c.sniffHttp() {
		return TypeHttp
	}

	// TODO: May need to check more stream types

	return TypeUnknown
}

func (c *SniffConn) peek(n int) ([]byte, error) {
	if c.read {
		return nil, errors.New("peek must before read")
	}
	if c.peeked {
		return nil, errors.New("can only peek once")
	}
	c.peeked = true
	peeks := make([]byte, n)
	n, err := c.Conn.Read(peeks)
	return peeks[:n], err
}

func (c *SniffConn) sniffHttp() bool {
	parts := bytes.Split(c.peeks, sep)
	if len(parts) < 2 {
		return false
	}
	for _, m := range httpMethods {
		if bytes.Compare(parts[0], m) == 0 {
			return true
		}
	}
	return false
}
