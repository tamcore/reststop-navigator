package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestMatchCountryByBBox(t *testing.T) {
	cases := []struct {
		name string
		q    string
		want string
		ok   bool
	}{
		{"de tile", `way["highway"="motorway"](48,11,48.5,11.5);`, "DE", true},
		// AT bbox is S:46.37 N:49.02; pick a tile whose centre is south of
		// DE's south edge (47.27) so DE doesn't claim it first.
		{"at tile", `way["highway"="motorway"](46.5,13,47.5,13.5);`, "AT", true},
		{"sk tile", `way["highway"="motorway"](48.5,18,49,18.5);`, "SK", true},
		// CZ bbox is S:48.55 W:12.09 N:51.06 E:18.86 — pick a tile firmly
		// inside it and outside DE/AT/SK.
		{"cz tile", `way["highway"="motorway"](49.5,15,50,15.5);`, "CZ", true},
		{"no tuple", `// nothing here`, "", false},
		{"tuple outside any country", `way["highway"="motorway"](0,0,1,1);`, "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := matchCountryByBBox(c.q)
			if ok != c.ok || got != c.want {
				t.Fatalf("matchCountryByBBox(%q) = (%q, %v), want (%q, %v)", c.q, got, ok, c.want, c.ok)
			}
		})
	}
}

func TestDecodeQuery(t *testing.T) {
	form := url.Values{"data": {"way(48,11,48.5,11.5);"}}
	got := decodeQuery([]byte(form.Encode()))
	if got != "way(48,11,48.5,11.5);" {
		t.Fatalf("decodeQuery form: got %q", got)
	}

	// Non form-encoded body falls through unchanged.
	raw := "raw overpass body"
	if decodeQuery([]byte(raw)) != raw {
		t.Fatalf("decodeQuery raw: got %q", decodeQuery([]byte(raw)))
	}
}

func TestHandleServesDEFixtureForBBox(t *testing.T) {
	form := url.Values{"data": {`way["highway"="motorway"](48,11,48.5,11.5);`}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handle(w, req)

	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"ref": "A8"`) {
		t.Fatalf("expected DE fixture (A8) in response, got: %s", string(body))
	}
}

func TestHandleLegacyAreaFilter(t *testing.T) {
	q := `area["ISO3166-1"="DE"]; way(area);`
	form := url.Values{"data": {q}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	w := httptest.NewRecorder()
	handle(w, req)
	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), `"ref": "A8"`) {
		t.Fatalf("expected DE fixture for legacy area filter, got: %s", string(body))
	}
}

func TestHandleUnknownReturnsEmpty(t *testing.T) {
	form := url.Values{"data": {"// nothing useful"}}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	w := httptest.NewRecorder()
	handle(w, req)
	resp := w.Result()
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if strings.TrimSpace(string(body)) != `{"elements":[]}` {
		t.Fatalf("expected empty elements, got: %s", string(body))
	}
}

func TestHandleRejectsGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handle(w, req)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", w.Result().StatusCode)
	}
}
