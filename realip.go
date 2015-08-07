// Package realip mimics the nginx realip module.
// It rewrites RemoteAddr in an http.Request when the
// connection is from a trusted network.
package realip

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

var errBadIP = errors.New("invalid ip address")

// RealIP is used to override RemoteAddr for net/http.
type RealIP struct {
	allowed []*net.IPNet
	headers []string
}

// New creates a new RealIP that uses the given headers and networks.
// The first matching header is used.
func New(headers []string, nets []string) (*RealIP, error) {
	ri := RealIP{
		headers: make([]string, len(headers)),
	}
	copy(ri.headers, headers)

	for _, i := range nets {
		_, net, err := net.ParseCIDR(i)
		if err != nil {
			return nil, err
		}
		ri.allowed = append(ri.allowed, net)
	}

	return &ri, nil
}

func parseIP(val string) string {
	for _, ip := range strings.Split(val, ",") {
		ip = strings.TrimSpace(ip)
		if v := net.ParseIP(ip); v != nil {
			return ip
		}
	}
	return ""
}

func (ri *RealIP) getHeader(r *http.Request) string {
	for _, h := range ri.headers {
		if v := r.Header.Get(h); v != "" {
			return v
		}
	}
	return ""
}

func getRemoteAddress(r *http.Request) (net.IP, string, error) {
	address, port, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, "", err
	}
	ip := net.ParseIP(address)
	if ip == nil {
		return nil, "", errBadIP
	}
	return ip, port, nil
}

func (ri *RealIP) getRealIPValue(r *http.Request) string {

	header := ri.getHeader(r)
	if header == "" {
		return ""
	}
	return parseIP(header)
}

// Handler wraps another Handler.
func (ri *RealIP) Handler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, port, err := getRemoteAddress(r)
		if err != nil {
			// should this be fatal or should we just skip?
			http.Error(w, "invalid remote address", 400)
			return
		}

		//O(n) - could do a better way, but this handles v6, etc
		for _, n := range ri.allowed {
			if n.Contains(ip) {
				val := ri.getRealIPValue(r)
				if val != "" {
					r.RemoteAddr = net.JoinHostPort(val, port)
				}
				break
			}
		}
		h.ServeHTTP(w, r)
	})
}
