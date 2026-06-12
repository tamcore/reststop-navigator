package middleware

import (
	"log/slog"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// AdminAuth guards admin endpoints with HTTP Basic Auth. Only the password is
// checked; the username is ignored. An empty configured password rejects all
// requests — callers should not mount admin routes at all in that case, this
// is just defense in depth.
//
// The configured password is bcrypt-hashed once at construction and each
// request is verified with bcrypt's constant-time comparison. The ~50-100ms
// per request is irrelevant for a single-user admin surface and doubles as
// brute-force throttling.
func AdminAuth(password string) func(http.Handler) http.Handler {
	var hash []byte
	if password != "" {
		var err error
		hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			// Only reachable for passwords >72 bytes; treat as misconfiguration
			// and reject everything rather than failing open.
			slog.Error("admin auth: bcrypt hash failed, admin access disabled", "error", err)
			hash = nil
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, got, ok := r.BasicAuth()
			if hash == nil || !ok || bcrypt.CompareHashAndPassword(hash, []byte(got)) != nil {
				w.Header().Set("WWW-Authenticate", `Basic realm="reststop-admin"`)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
