package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tamcore/reststop-navigator/internal/api"
)

func TestHealthzReturnsOK(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(api.NewRouter())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/healthz")
	if err != nil {
		t.Fatalf("GET /api/healthz: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got, want := resp.Header.Get("Content-Type"), "application/json"; got != want {
		t.Errorf("content-type = %q, want %q", got, want)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal body %q: %v", body, err)
	}
	if payload.Status != "ok" {
		t.Errorf("status field = %q, want %q", payload.Status, "ok")
	}
}

func TestUnknownRouteReturns404(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(api.NewRouter())
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/does-not-exist")
	if err != nil {
		t.Fatalf("GET unknown: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}
