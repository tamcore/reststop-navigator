package middleware

import (
	"net"
	"net/http"
	"strings"
)

// RealIP sets r.RemoteAddr to the client IP reported by the nearest trusted
// reverse proxy. It reads X-Real-IP first (set by NGINX ingress to the actual
// client IP), then falls back to the rightmost non-private value in
// X-Forwarded-For (the entry appended by the closest upstream, which the
// client cannot inject).
//
// Unlike chimw.RealIP, this implementation only rewrites RemoteAddr when the
// connection originates from a private/loopback address, ensuring that a
// directly-connected public client cannot override its own source address.
func RealIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ip := realClientIP(r); ip != "" {
			r = r.WithContext(r.Context())
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	})
}

func realClientIP(r *http.Request) string {
	connIP, _, _ := net.SplitHostPort(r.RemoteAddr)
	if connIP == "" {
		connIP = r.RemoteAddr
	}
	if !isPrivateOrLoopback(connIP) {
		return ""
	}
	if v := r.Header.Get("X-Real-IP"); v != "" {
		if net.ParseIP(strings.TrimSpace(v)) != nil {
			return v
		}
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			candidate := strings.TrimSpace(parts[i])
			if ip := net.ParseIP(candidate); ip != nil {
				return candidate
			}
		}
	}
	return ""
}

var privateRanges = []net.IPNet{
	parseCIDR("10.0.0.0/8"),
	parseCIDR("172.16.0.0/12"),
	parseCIDR("192.168.0.0/16"),
	parseCIDR("fc00::/7"),
	parseCIDR("127.0.0.0/8"),
	parseCIDR("::1/128"),
}

func isPrivateOrLoopback(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	for _, r := range privateRanges {
		if r.Contains(ip) {
			return true
		}
	}
	return false
}

func parseCIDR(s string) net.IPNet {
	_, n, _ := net.ParseCIDR(s)
	return *n
}
