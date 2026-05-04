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
		// 'unsafe-inline' for script-src is required by SvelteKit's static adapter,
		// which emits inline hydration data scripts. Acceptable for an unauthenticated
		// PWA with no per-user state; revisit if XSS surface grows.
		// img-src includes the OSM tile servers used by the detail-page map
		// (leaflet pulls tiles from {a,b,c}.tile.openstreetmap.org).
		// font-src 'self' data: is required by @fontsource — its CSS inlines
		// glyph subsets as data: URIs. Without it the page silently falls
		// back to the system serif and the design language collapses.
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; img-src 'self' data: https://*.tile.openstreetmap.org https://tile.openstreetmap.org; style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self'; manifest-src 'self'")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
