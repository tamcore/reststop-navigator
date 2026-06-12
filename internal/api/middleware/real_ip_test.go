package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/api/middleware"
)

func TestRealIP(t *testing.T) {
	t.Parallel()

	captureAddr := func() (http.Handler, *string) {
		var addr string
		h := middleware.RealIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			addr = r.RemoteAddr
		}))
		return h, &addr
	}

	tests := []struct {
		name       string
		remoteAddr string
		xRealIP    string
		xff        string
		wantAddr   string
	}{
		{
			name:       "direct public client is not overridden",
			remoteAddr: "1.2.3.4:12345",
			xRealIP:    "9.9.9.9",
			wantAddr:   "1.2.3.4:12345",
		},
		{
			name:       "X-Real-IP used when request comes from private IP",
			remoteAddr: "10.0.0.1:56789",
			xRealIP:    "203.0.113.7",
			wantAddr:   "203.0.113.7",
		},
		{
			name:       "X-Forwarded-For rightmost used when no X-Real-IP",
			remoteAddr: "172.16.0.5:12345",
			xff:        "203.0.113.1, 10.0.0.2",
			wantAddr:   "10.0.0.2",
		},
		{
			name:       "X-Real-IP preferred over X-Forwarded-For",
			remoteAddr: "192.168.1.1:9090",
			xRealIP:    "203.0.113.99",
			xff:        "203.0.113.1, 10.0.0.2",
			wantAddr:   "203.0.113.99",
		},
		{
			name:       "loopback treated as trusted proxy",
			remoteAddr: "127.0.0.1:1234",
			xRealIP:    "1.1.1.1",
			wantAddr:   "1.1.1.1",
		},
		{
			name:       "invalid X-Real-IP is ignored, falls through to XFF",
			remoteAddr: "10.0.0.1:1234",
			xRealIP:    "not-an-ip",
			xff:        "5.6.7.8",
			wantAddr:   "5.6.7.8",
		},
		{
			name:       "no headers leaves RemoteAddr unchanged",
			remoteAddr: "10.0.0.1:1234",
			wantAddr:   "10.0.0.1:1234",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h, addr := captureAddr()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			if tc.xff != "" {
				req.Header.Set("X-Forwarded-For", tc.xff)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			if *addr != tc.wantAddr {
				t.Errorf("RemoteAddr = %q, want %q", *addr, tc.wantAddr)
			}
		})
	}
}
