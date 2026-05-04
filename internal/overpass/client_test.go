package overpass_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tamcore/reststop-navigator/internal/overpass"
)

func newServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestClient_QuerySuccess(t *testing.T) {
	t.Parallel()

	var hits int64
	srv := newServer(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "data=") {
			t.Errorf("expected form-encoded data, got %q", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"elements":[]}`))
	})
	t.Cleanup(srv.Close)

	c := overpass.NewClient([]string{srv.URL},
		overpass.WithBackoff(time.Millisecond, 4*time.Millisecond),
		overpass.WithMaxRetries(2),
	)
	got, err := c.Query(context.Background(), "out;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != `{"elements":[]}` {
		t.Errorf("body = %q", got)
	}
	if atomic.LoadInt64(&hits) != 1 {
		t.Errorf("hits = %d, want 1", hits)
	}
}

func TestClient_RetriesOn5xx(t *testing.T) {
	t.Parallel()

	var hits int64
	srv := newServer(func(w http.ResponseWriter, _ *http.Request) {
		n := atomic.AddInt64(&hits, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok-after-retries"))
	})
	t.Cleanup(srv.Close)

	c := overpass.NewClient([]string{srv.URL},
		overpass.WithBackoff(time.Millisecond, 4*time.Millisecond),
		overpass.WithMaxRetries(5),
	)
	got, err := c.Query(context.Background(), "out;")
	if err != nil {
		t.Fatalf("unexpected error after retries: %v", err)
	}
	if string(got) != "ok-after-retries" {
		t.Errorf("body = %q", got)
	}
	if h := atomic.LoadInt64(&hits); h != 3 {
		t.Errorf("hits = %d, want 3", h)
	}
}

func TestClient_FallsBackToNextEndpoint(t *testing.T) {
	t.Parallel()

	var primaryHits, fallbackHits int64
	primary := newServer(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&primaryHits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	t.Cleanup(primary.Close)
	fallback := newServer(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&fallbackHits, 1)
		_, _ = w.Write([]byte("from-fallback"))
	})
	t.Cleanup(fallback.Close)

	c := overpass.NewClient([]string{primary.URL, fallback.URL},
		overpass.WithBackoff(time.Millisecond, 2*time.Millisecond),
		overpass.WithMaxRetries(1),
	)
	got, err := c.Query(context.Background(), "out;")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(got) != "from-fallback" {
		t.Errorf("body = %q", got)
	}
	if atomic.LoadInt64(&primaryHits) < 1 {
		t.Error("expected primary to be called at least once")
	}
	if atomic.LoadInt64(&fallbackHits) != 1 {
		t.Errorf("fallback hits = %d, want 1", fallbackHits)
	}
}

func TestClient_AllEndpointsFail(t *testing.T) {
	t.Parallel()

	srv := newServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	t.Cleanup(srv.Close)

	c := overpass.NewClient([]string{srv.URL, srv.URL},
		overpass.WithBackoff(time.Millisecond, 2*time.Millisecond),
		overpass.WithMaxRetries(1),
	)
	if _, err := c.Query(context.Background(), "out;"); err == nil {
		t.Fatal("expected error when all endpoints fail")
	}
}

func TestClient_NonRetryableStatusFailsImmediately(t *testing.T) {
	t.Parallel()

	var hits int64
	srv := newServer(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&hits, 1)
		w.WriteHeader(http.StatusBadRequest)
	})
	t.Cleanup(srv.Close)

	c := overpass.NewClient([]string{srv.URL},
		overpass.WithBackoff(time.Millisecond, 2*time.Millisecond),
		overpass.WithMaxRetries(5),
	)
	if _, err := c.Query(context.Background(), "out;"); err == nil {
		t.Fatal("expected error on 400")
	}
	if h := atomic.LoadInt64(&hits); h != 1 {
		t.Errorf("400 should not retry: hits = %d, want 1", h)
	}
}

func TestClient_ContextCancelStopsRetries(t *testing.T) {
	t.Parallel()

	srv := newServer(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	c := overpass.NewClient([]string{srv.URL},
		overpass.WithBackoff(50*time.Millisecond, 50*time.Millisecond),
		overpass.WithMaxRetries(20),
	)

	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	if _, err := c.Query(ctx, "out;"); err == nil {
		t.Fatal("expected error from context cancel")
	}
}

func TestClient_NoEndpointsConfigured(t *testing.T) {
	t.Parallel()

	c := overpass.NewClient(nil)
	if _, err := c.Query(context.Background(), "out;"); err == nil {
		t.Fatal("expected error with no endpoints")
	}
}
