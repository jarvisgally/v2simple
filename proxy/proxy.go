package proxy

import (
	"errors"
	"io"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// Client is used to create connection.
type Client interface {
	Name() string
	Addr() string
	Handshake(underlay net.Conn, target string) (io.ReadWriter, error)
}

// ClientCreator is a function to create client.
type ClientCreator func(url *url.URL) (Client, error)

var (
	clientMap = make(map[string]ClientCreator)
)

// RegisterClient is used to register a client.
func RegisterClient(name string, c ClientCreator) {
	clientMap[name] = c
}

// ClientFromURL calls the registered creator to create client.
// dialer is the default upstream dialer so cannot be nil, we can use Default when calling this function.
func ClientFromURL(s string) (Client, error) {
	u, err := url.Parse(s)
	if err != nil {
		log.Printf("can not parse client url %s err: %s", s, err)
		return nil, err
	}

	c, ok := clientMap[strings.ToLower(u.Scheme)]
	if ok {
		return c(u)
	}

	return nil, errors.New("unknown client scheme '" + u.Scheme + "'")
}

// Server interface
type Server interface {
	Name() string
	Addr() string
	Handshake(underlay net.Conn) (io.ReadWriter, *TargetAddr, error)
	Stop()
}

// ServerCreator is a function to create proxy server
type ServerCreator func(url *url.URL) (Server, error)

var (
	serverMap = make(map[string]ServerCreator)
)

// RegisterServer is used to register a proxy server
func RegisterServer(name string, c ServerCreator) {
	serverMap[name] = c
}

// ServerFromURL calls the registered creator to create proxy servers
// dialer is the default upstream dialer so cannot be nil, we can use Default when calling this function
func ServerFromURL(s string) (Server, error) {
	u, err := url.Parse(s)
	if err != nil {
		log.Printf("can not parse server url %s err: %s", s, err)
		return nil, err
	}

	c, ok := serverMap[strings.ToLower(u.Scheme)]
	if ok {
		return c(u)
	}

	return nil, errors.New("unknown server scheme '" + u.Scheme + "'")
}

// An Addr represents a address that you want to access by proxy. Either Name or IP is used exclusively.
type TargetAddr struct {
	Name string // fully-qualified domain name
	IP   net.IP
	Port int
}

// Return host:port string
func (a *TargetAddr) String() string {
	port := strconv.Itoa(a.Port)
	if a.IP == nil {
		return net.JoinHostPort(a.Name, port)
	}
	return net.JoinHostPort(a.IP.String(), port)
}

// Returned host string
func (a *TargetAddr) Host() string {
	if a.IP == nil {
		return a.Name
	}
	return a.IP.String()
}

func NewTargetAddr(addr string) (*TargetAddr, error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if host == "" {
		host = "127.0.0.1"
	}
	port, err := strconv.Atoi(portStr)

	target := &TargetAddr{Port: port}
	if ip := net.ParseIP(host); ip != nil {
		target.IP = ip
	} else {
		target.Name = host
	}
	return target, nil
}

