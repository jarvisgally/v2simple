package tls

import (
	stdtls "crypto/tls"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"

	"github.com/jarvisgally/v2simple/proxy"
)

func init() {
	proxy.RegisterServer("vmesss", NewTlsServer)
}

func NewTlsServer(url *url.URL) (proxy.Server, error) {
	addr := url.Host
	sni, _, _ := net.SplitHostPort(addr)

	query := url.Query()
	certFile := query.Get("cert")
	keyFile := query.Get("key")
	cert, err := stdtls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	fallback := query.Get("fallback")
	if fallback != "" {
		_, _, err = net.SplitHostPort(fallback)
		if err != nil {
			return nil, err
		}
	}

	s := &Server{name: url.Scheme, addr: addr, fallback: fallback}
	s.tlsConfig = &stdtls.Config{
		InsecureSkipVerify: false,
		ServerName:         sni,
		Certificates:       []stdtls.Certificate{cert},
	}

	url.Scheme = strings.TrimSuffix(url.Scheme, "s")
	s.inner, _ = proxy.ServerFromURL(url.String())

	return s, nil
}

type Server struct {
	name      string
	addr      string
	fallback  string
	tlsConfig *stdtls.Config

	inner proxy.Server
}

func (s *Server) Name() string { return s.name }

func (s *Server) Addr() string { return s.addr }

func (s *Server) Handshake(underlay net.Conn) (io.ReadWriter, *proxy.TargetAddr, error) {
	ss := stdtls.Server(underlay, s.tlsConfig)
	err := ss.Handshake()
	if err != nil {
		return nil, nil, errors.New("invalid handshake")
	}
	// TODO: Check if a http request, redirect to fallback address
	return s.inner.Handshake(ss)
}

func (s *Server) Stop() {
	s.inner.Stop()
}
