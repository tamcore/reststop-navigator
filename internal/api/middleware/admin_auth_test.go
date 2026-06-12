package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/api/middleware"
)

func adminProtected(password string) http.Handler {
	return middleware.AdminAuth(password)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
}

func TestAdminAuth_RejectsMissingCredentials(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	rec := httptest.NewRecorder()
	adminProtected("s3cret").ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if got := rec.Header().Get("WWW-Authenticate"); got != `Basic realm="reststop-admin"` {
		t.Errorf("WWW-Authenticate = %q", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestAdminAuth_RejectsWrongPassword(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	req.SetBasicAuth("admin", "wrong")
	rec := httptest.NewRecorder()
	adminProtected("s3cret").ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAdminAuth_AcceptsCorrectPasswordAnyUsername(t *testing.T) {
	t.Parallel()

	for _, user := range []string{"admin", "", "whatever"} {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
		req.SetBasicAuth(user, "s3cret")
		rec := httptest.NewRecorder()
		adminProtected("s3cret").ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("user %q: status = %d, want 200", user, rec.Code)
		}
	}
}

func TestAdminAuth_RejectsEverythingWhenPasswordEmpty(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/api/admin/stats", nil)
	req.SetBasicAuth("admin", "")
	rec := httptest.NewRecorder()
	adminProtected("").ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (empty configured password must never authenticate)", rec.Code)
	}
}
