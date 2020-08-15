package common

import (
	"bufio"
	"net"
	"os"
	"strings"

	"github.com/yl2chen/cidranger"
)

type Matcher struct {
	NetRanger cidranger.Ranger
	IPMap     map[string]net.IP
	DomainMap map[string]string
}

// New a matcher that checks if a cidr or ip or domain is in a predefined list.
func NewMather(configFileName string) *Matcher {
	// Use cidranger to match net.IPNet
	// https://github.com/yl2chen/cidranger
	ranger := cidranger.NewPCTrieRanger()
	// Use map to match specified ip or domain
	ipMap := make(map[string]net.IP)
	domainMap := make(map[string]string)

	// Parse config file
	path := GetPath(configFileName)
	if len(path) > 0 {
		if cf, err := os.Open(path); err == nil {
			defer cf.Close()
			scanner := bufio.NewScanner(cf)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if len(line) == 0 {
					continue
				}
				if strings.Contains(line, "/") {
					if _, net, err := net.ParseCIDR(line); err == nil {
						ranger.Insert(cidranger.NewBasicRangerEntry(*net))
					}
					continue
				}
				if ip := net.ParseIP(line); ip != nil {
					ipMap[line] = ip
					continue
				}
				domainMap[line] = line
			}
		}
	}

	return &Matcher{
		NetRanger: ranger,
		IPMap:     ipMap,
		DomainMap: domainMap,
	}
}

func (m *Matcher) Check(host string) bool {
	if ip := net.ParseIP(host); ip != nil {
		if m.NetRanger != nil {
			if contains, _ := m.NetRanger.Contains(ip); contains {
				return true
			}
		}
		if m.IPMap != nil {
			if _, found := m.IPMap[host]; found {
				return true
			}
		}
	}
	tokens := strings.Split(host, ".")
	if len(tokens) > 1 {
		suffix := tokens[len(tokens)-1]
		for i := len(tokens) - 2; i >= 0; i-- {
			suffix = tokens[i] + "." + suffix
			if _, found := m.DomainMap[suffix]; found {
				return true
			}
		}
	}
	return false
}
