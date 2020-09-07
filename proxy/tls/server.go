package tls

import (
	stdtls "crypto/tls"
	"errors"
	"fmt"
	"github.com/jarvisgally/v2simple/common"
	"io"
	"log"
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
	var fallbackAddr *proxy.TargetAddr
	if fallback != "" {
		fallbackAddr, err = proxy.NewTargetAddr(fallback)
		if err != nil {
			return nil, fmt.Errorf("invalid fallback %v", fallbackAddr)
		}
	}

	s := &Server{name: url.Scheme, addr: addr, fallbackAddr: fallbackAddr}
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
	fallbackAddr *proxy.TargetAddr
	tlsConfig *stdtls.Config

	inner proxy.Server
}

func (s *Server) Name() string { return s.name }

func (s *Server) Addr() string { return s.addr }

func (s *Server) Handshake(underlay net.Conn) (io.ReadWriter, *proxy.TargetAddr, error) {
	tlsConn := stdtls.Server(underlay, s.tlsConfig)
	err := tlsConn.Handshake()
	if err != nil {
		return nil, nil, errors.New("invalid handshake")
	}
	sniffConn := common.NewSniffConn(tlsConn)
	t := sniffConn.Sniff()
	if t == common.TypeUnknown {
		// this is not a http request, route to inner protocol, e.g, vmess/trojan
		return s.inner.Handshake(sniffConn)
	} else {
		// http request, route to fallback address
		if s.fallbackAddr != nil {
			log.Printf("http request, redirect to %v", s.fallbackAddr)
			return sniffConn, s.fallbackAddr, nil
		} else {
			return nil, nil, errors.New("not supported")
		}
	}
}

func (s *Server) Stop() {
	s.inner.Stop()
}
