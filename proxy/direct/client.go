package direct

import (
	"io"
	"net"
	"net/url"

	"github.com/jarvisgally/v2simple/proxy"
)

const name = "direct"

func init() {
	proxy.RegisterClient(name, NewDirectClient)
}

func NewDirectClient(url *url.URL) (proxy.Client, error) {
	return &Direct{}, nil
}

type Direct struct{}

func (d *Direct) Name() string { return name }

func (d *Direct) Addr() string { return name }

func (d *Direct) Handshake(underlay net.Conn, target string) (io.ReadWriter, error) {
	return underlay, nil
}
