// Package middleware contains HTTP middlewares used by internal/api/router.
package middleware

import "net/http"

// SecurityHeaders sets a defensive default header set on every response:
// nosniff, DENY-frame, restrictive Referrer-Policy, geolocation-only
// Permissions-Policy, and a tight Content-Security-Policy. Strict-Transport-
// Security is added only on TLS-backed requests so plain-HTTP local dev still
// works.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "geolocation=(self)")
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; connect-src 'self'; manifest-src 'self'")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
