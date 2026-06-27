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
		// strict-origin-when-cross-origin (not no-referrer): the OSM tile
		// usage policy requires a valid Referer on tile requests, and this
		// policy sends only the origin cross-origin — no path or query.
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(self)")
		// 'unsafe-inline' for script-src is required by SvelteKit's static adapter,
		// which emits inline hydration data scripts. Acceptable for an unauthenticated
		// PWA with no per-user state; revisit if XSS surface grows.
		// img-src includes the BKG TopPlusOpen tile server (EU-hosted, German
		// federal) used by the detail-page and admin maps. OSM raster tiles are
		// fronted by a US CDN, so we don't allowlist them.
		// font-src 'self' data: is required by @fontsource — its CSS inlines
		// glyph subsets as data: URIs. Without it the page silently falls
		// back to the system serif and the design language collapses.
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; img-src 'self' data: https://sgx.geodatenzentrum.de; style-src 'self' 'unsafe-inline'; font-src 'self' data:; connect-src 'self'; manifest-src 'self'")
		if r.TLS != nil {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
