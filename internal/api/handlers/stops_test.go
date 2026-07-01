package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/tamcore/reststop-navigator/internal/api/handlers"
	"github.com/tamcore/reststop-navigator/internal/geo"
	"github.com/tamcore/reststop-navigator/internal/overpass"
	"github.com/tamcore/reststop-navigator/internal/presence"
	"github.com/tamcore/reststop-navigator/internal/stops"
)

type fakeService struct {
	upcoming     func(stops.UpcomingRequest) (stops.UpcomingResponse, error)
	get          func(string) (stops.DetailResponse, error)
	gotRequest   *stops.UpcomingRequest
	gotDetailID  string
	gotDetailPos geo.LatLng
}

func (f *fakeService) Upcoming(_ context.Context, req stops.UpcomingRequest) (stops.UpcomingResponse, error) {
	f.gotRequest = &req
	if f.upcoming != nil {
		return f.upcoming(req)
	}
	return stops.UpcomingResponse{Stops: []stops.StopInfo{}}, nil
}

func (f *fakeService) Get(_ context.Context, id string, pos geo.LatLng) (stops.DetailResponse, error) {
	f.gotDetailID = id
	f.gotDetailPos = pos
	if f.get != nil {
		return f.get(id)
	}
	return stops.DetailResponse{}, stops.ErrStopNotFound
}

type recordedPosition struct {
	clientID string
	pos      presence.Position
}

type fakeRecorder struct {
	ch chan recordedPosition
}

func newFakeRecorder() *fakeRecorder {
	return &fakeRecorder{ch: make(chan recordedPosition, 8)}
}

func (f *fakeRecorder) Record(_ context.Context, clientID string, p presence.Position) error {
	f.ch <- recordedPosition{clientID: clientID, pos: p}
	return nil
}

func mountServer(h *handlers.Stops) *httptest.Server {
	r := chi.NewRouter()
	h.Mount(r)
	return httptest.NewServer(r)
}

func TestUpcoming_RequiresLatLon(t *testing.T) {
	t.Parallel()
	srv := mountServer(handlers.NewStops(&fakeService{}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/upcoming?heading=90")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUpcoming_ParsesQuery(t *testing.T) {
	t.Parallel()
	fs := &fakeService{
		upcoming: func(_ stops.UpcomingRequest) (stops.UpcomingResponse, error) {
			return stops.UpcomingResponse{
				Country: "DE",
				Road:    &stops.Road{Ref: "A8", Direction: "E"},
				Stops: []stops.StopInfo{
					{ID: "node/1", Kind: "services", Lat: 48, Lon: 11.005, DistanceM: 100},
				},
				Version: "1714824000",
			}, nil
		},
	}
	srv := mountServer(handlers.NewStops(fs))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/upcoming?lat=48.13&lon=11.58&heading=90&speed=120&filters=fuel,charging&limit=5")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	if fs.gotRequest.Pos.Lat != 48.13 || fs.gotRequest.Pos.Lon != 11.58 {
		t.Errorf("pos = %+v", fs.gotRequest.Pos)
	}
	if fs.gotRequest.Heading != 90 {
		t.Errorf("heading = %v", fs.gotRequest.Heading)
	}
	if fs.gotRequest.Speed != 120 {
		t.Errorf("speed = %v", fs.gotRequest.Speed)
	}
	if !fs.gotRequest.Filters.Fuel || !fs.gotRequest.Filters.Charging {
		t.Errorf("filters = %+v", fs.gotRequest.Filters)
	}
	if fs.gotRequest.Filters.Food {
		t.Errorf("food filter unset should not be true")
	}
	if fs.gotRequest.Limit != 5 {
		t.Errorf("limit = %v", fs.gotRequest.Limit)
	}

	var body stops.UpcomingResponse
	mustDecode(t, resp.Body, &body)
	if body.Country != "DE" || body.Road.Ref != "A8" {
		t.Errorf("body = %+v", body)
	}
}

func TestUpcoming_ServiceError500(t *testing.T) {
	t.Parallel()
	fs := &fakeService{
		upcoming: func(_ stops.UpcomingRequest) (stops.UpcomingResponse, error) {
			return stops.UpcomingResponse{}, errors.New("redis is down")
		},
	}
	srv := mountServer(handlers.NewStops(fs))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/upcoming?lat=48&lon=11&heading=90")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
}

func TestUpcoming_BadFloats400(t *testing.T) {
	t.Parallel()
	srv := mountServer(handlers.NewStops(&fakeService{}))
	t.Cleanup(srv.Close)

	for _, q := range []string{
		"lat=abc&lon=11&heading=90",
		"lat=48&lon=xyz&heading=90",
		"lat=48&lon=11&heading=hi",
	} {
		resp, err := http.Get(srv.URL + "/api/stops/upcoming?" + q)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("query %q -> status %d, want 400", q, resp.StatusCode)
		}
	}
}

func TestDetail_HappyPath(t *testing.T) {
	t.Parallel()
	fs := &fakeService{
		get: func(id string) (stops.DetailResponse, error) {
			return stops.DetailResponse{
				Country: "DE",
				Stop: stops.StopInfo{
					ID:   id,
					Name: "Aichen",
					Kind: "services",
					Lat:  48,
					Lon:  11.005,
					Amenities: overpass.AmenityFlags{
						Fuel: true, Food: true, Toilets: true, Open24h: true,
					},
				},
				DeepLinks: stops.DeepLinks{
					Google: "https://www.google.com/maps/dir/?api=1&destination=48,11.005",
					Apple:  "https://maps.apple.com/?daddr=48,11.005",
					Waze:   "https://waze.com/ul?ll=48,11.005&navigate=yes",
				},
			}, nil
		},
	}
	srv := mountServer(handlers.NewStops(fs))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/detail?id=node/100&lat=48&lon=11.005")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if fs.gotDetailID != "node/100" {
		t.Errorf("got id = %q", fs.gotDetailID)
	}

	var body stops.DetailResponse
	mustDecode(t, resp.Body, &body)
	if body.Stop.Name != "Aichen" {
		t.Errorf("body = %+v", body)
	}
	if !strings.Contains(body.DeepLinks.Google, "google.com") {
		t.Errorf("google link missing: %q", body.DeepLinks.Google)
	}
}

func TestDetail_NotFound404(t *testing.T) {
	t.Parallel()
	srv := mountServer(handlers.NewStops(&fakeService{}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/detail?id=node/missing&lat=48&lon=11.005")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestDetail_RequiresID400(t *testing.T) {
	t.Parallel()
	srv := mountServer(handlers.NewStops(&fakeService{}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/detail")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func mustDecode(t *testing.T, r io.Reader, v any) {
	t.Helper()
	body, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, v); err != nil {
		t.Fatalf("decode body %q: %v", body, err)
	}
}

func TestUpcoming_ParsesAccuracy(t *testing.T) {
	t.Parallel()
	fs := &fakeService{}
	srv := mountServer(handlers.NewStops(fs))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/upcoming?lat=48&lon=11&heading=90&accuracy=150")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if fs.gotRequest.Accuracy != 150 {
		t.Errorf("accuracy = %v, want 150", fs.gotRequest.Accuracy)
	}
}

func TestUpcoming_AccuracyDefaultsToZero(t *testing.T) {
	t.Parallel()
	fs := &fakeService{}
	srv := mountServer(handlers.NewStops(fs))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/upcoming?lat=48&lon=11&heading=90")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if fs.gotRequest.Accuracy != 0 {
		t.Errorf("accuracy = %v, want 0 (default)", fs.gotRequest.Accuracy)
	}
}

func TestUpcoming_AccuracyBadValue400(t *testing.T) {
	t.Parallel()
	srv := mountServer(handlers.NewStops(&fakeService{}))
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/api/stops/upcoming?lat=48&lon=11&heading=90&accuracy=abc")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestUpcoming_RecordsPresenceForValidClientID(t *testing.T) {
	t.Parallel()
	rec := newFakeRecorder()
	srv := mountServer(handlers.NewStops(&fakeService{}, handlers.WithPresenceRecorder(rec)))
	t.Cleanup(srv.Close)

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/stops/upcoming?lat=48.1&lon=11.5&heading=92.5&speed=104&accuracy=8", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Client-Id", "b3a4c1d2-5e6f-4a7b-8c9d-0e1f2a3b4c5d")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}

	select {
	case got := <-rec.ch:
		if got.clientID != "b3a4c1d2-5e6f-4a7b-8c9d-0e1f2a3b4c5d" {
			t.Errorf("clientID = %q", got.clientID)
		}
		if got.pos.Lat != 48.1 || got.pos.Lon != 11.5 || got.pos.Heading != 92.5 || got.pos.Speed != 104 || got.pos.Accuracy != 8 {
			t.Errorf("position = %+v", got.pos)
		}
		if got.pos.LastSeen.IsZero() {
			t.Error("LastSeen should be set")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("presence recorder was not called")
	}
}

func TestUpcoming_SkipsPresenceForInvalidOrMissingClientID(t *testing.T) {
	t.Parallel()
	rec := newFakeRecorder()
	srv := mountServer(handlers.NewStops(&fakeService{}, handlers.WithPresenceRecorder(rec)))
	t.Cleanup(srv.Close)

	for _, id := range []string{"", "not-a-uuid"} {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/stops/upcoming?lat=48&lon=11", nil)
		if err != nil {
			t.Fatal(err)
		}
		if id != "" {
			req.Header.Set("X-Client-Id", id)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d", resp.StatusCode)
		}
	}

	select {
	case got := <-rec.ch:
		t.Fatalf("recorder called unexpectedly with %+v", got)
	case <-time.After(100 * time.Millisecond):
	}
}

// guard against signature drift — the handler's service interface must accept
// the test's fake.
var _ handlers.StopsService = (*fakeService)(nil)
