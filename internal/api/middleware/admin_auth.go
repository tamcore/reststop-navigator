package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
)

// AdminAuth guards admin endpoints with HTTP Basic Auth. Only the password is
// checked; the username is ignored. An empty configured password rejects all
// requests — callers should not mount admin routes at all in that case, this
// is just defense in depth.
func AdminAuth(password string) func(http.Handler) http.Handler {
	want := sha256.Sum256([]byte(password))
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, got, ok := r.BasicAuth()
			// Hashing both sides makes ConstantTimeCompare applicable to
			// differing lengths and avoids leaking the password length.
			gotSum := sha256.Sum256([]byte(got))
			if password == "" || !ok || subtle.ConstantTimeCompare(want[:], gotSum[:]) != 1 {
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
