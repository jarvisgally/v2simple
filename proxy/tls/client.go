package tls

import (
	stdtls "crypto/tls"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/jarvisgally/v2simple/proxy"
)

func init() {
	proxy.RegisterClient("vmesss", NewTlsClient)
}

func NewTlsClient(url *url.URL) (proxy.Client, error) {
	addr := url.Host
	sni, _, _ := net.SplitHostPort(addr)

	c := &Client{name: url.Scheme, addr: addr}
	c.tlsConfig = &stdtls.Config{
		InsecureSkipVerify: true,
		ServerName:         sni,
	}

	url.Scheme = strings.TrimSuffix(url.Scheme, "s")
	c.inner, _ = proxy.ClientFromURL(url.String())

	return c, nil
}

type Client struct {
	name      string
	addr      string
	tlsConfig *stdtls.Config

	inner proxy.Client
}

func (c *Client) Name() string { return c.name }

func (c *Client) Addr() string { return c.addr }

func (c *Client) Handshake(underlay net.Conn, target string) (io.ReadWriter, error) {
	cc := stdtls.Client(underlay, c.tlsConfig)
	err := cc.Handshake()
	if err != nil {
		return nil, err
	}

	return c.inner.Handshake(cc, target)
}
