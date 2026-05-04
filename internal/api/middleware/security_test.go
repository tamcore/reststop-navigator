package middleware_test

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/api/middleware"
)

func TestSecurityHeaders_AppliesAll(t *testing.T) {
	t.Parallel()

	handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Permissions-Policy":      "geolocation=(self)",
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline'; img-src 'self' data: https://*.tile.openstreetmap.org https://tile.openstreetmap.org; style-src 'self' 'unsafe-inline'; connect-src 'self'; manifest-src 'self'",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestSecurityHeaders_HSTSOnlyOverHTTPS(t *testing.T) {
	t.Parallel()

	handler := middleware.SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tlsReq := httptest.NewRequest(http.MethodGet, "https://restops.example/", nil)
	tlsReq.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, tlsReq)
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("expected Strict-Transport-Security header on TLS request")
	}

	plainReq := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, plainReq)
	if got := rec2.Header().Get("Strict-Transport-Security"); got != "" {
		t.Errorf("HSTS should be omitted on plain HTTP, got %q", got)
	}
}
