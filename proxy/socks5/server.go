package socks5

import (
	"errors"
	"fmt"
	"github.com/jarvisgally/v2simple/common"
	"io"
	"net"
	"net/url"
	"time"

	"github.com/jarvisgally/v2simple/proxy"
)

func init() {
	proxy.RegisterServer(Name, NewSocks5Server)
}

func NewSocks5Server(url *url.URL) (proxy.Server, error) {
	addr := url.Host

	// TODO: Support Auth
	user := url.User.Username()
	password, _ := url.User.Password()

	s := &Server{
		addr:     addr,
		user:     user,
		password: password,
	}
	return s, nil
}

type Server struct {
	addr     string
	user     string
	password string
}

func (s *Server) Name() string { return Name }

func (s *Server) Addr() string { return s.addr }

func (s *Server) Handshake(underlay net.Conn) (io.ReadWriter, *proxy.TargetAddr, error) {
	// Set handshake timeout 4 seconds
	if err := underlay.SetReadDeadline(time.Now().Add(time.Second * 4)); err != nil {
		return nil, nil, err
	}
	defer underlay.SetReadDeadline(time.Time{})

	// https://www.ietf.org/rfc/rfc1928.txt
	buf := common.GetBuffer(512)
	defer common.PutBuffer(buf)

	// Read hello message
	n, err := underlay.Read(buf)
	if err != nil || n == 0 {
		return nil, nil, fmt.Errorf("failed to read hello: %w", err)
	}
	version := buf[0]
	if version != Version5 {
		return nil, nil, fmt.Errorf("unsupported socks version %v", version)
	}

	// Write hello response
	// TODO: Support Auth
	_, err = underlay.Write([]byte{Version5, AuthNone})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write hello response: %w", err)
	}

	// Read command message
	n, err = underlay.Read(buf)
	if err != nil || n < 7 { // Shortest length is 7
		return nil, nil, fmt.Errorf("failed to read command: %w", err)
	}
	cmd := buf[1]
	if cmd != CmdConnect {
		return nil, nil, fmt.Errorf("unsuppoted command %v", cmd)
	}
	addr := &proxy.TargetAddr{}
	l := 2
	off := 4
	switch buf[3] {
	case ATypIP4:
		l += net.IPv4len
		addr.IP = make(net.IP, net.IPv4len)
	case ATypIP6:
		l += net.IPv6len
		addr.IP = make(net.IP, net.IPv6len)
	case ATypDomain:
		l += int(buf[4])
		off = 5
	default:
		return nil, nil, fmt.Errorf("unknown address type %v", buf[3])
	}

	if len(buf[off:]) < l {
		return nil, nil, errors.New("short command request")
	}
	if addr.IP != nil {
		copy(addr.IP, buf[off:])
	} else {
		addr.Name = string(buf[off : off+l-2])
	}
	addr.Port = int(buf[off+l-2])<<8 | int(buf[off+l-1])

	// Write command response
	_, err = underlay.Write([]byte{Version5, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to write command response: %w", err)
	}

	return underlay, addr, err
}

func (s *Server) Stop() {
	// Nothing to stop or close
}
