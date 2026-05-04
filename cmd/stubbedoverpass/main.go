// Command stubbedoverpass is a tiny HTTP server that mimics the public
// Overpass API by serving canned per-country JSON fixtures. Used by
// docker-compose.dev.yaml and the e2e CI job so the stack never depends on
// the real overpass-api.de.
package main

import (
	"embed"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed fixtures/*.json
var fixtures embed.FS

const defaultAddr = ":7000"

func main() {
	addr := os.Getenv("STUBBED_OVERPASS_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           http.HandlerFunc(handle),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("stubbed-overpass listening on %s", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	q := string(body)
	for _, iso := range []string{"DE", "AT", "SK", "CZ"} {
		needle := `"ISO3166-1"%3D%22` + iso + `%22`
		if strings.Contains(q, needle) {
			serveFixture(w, iso)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"elements":[]}`))
}

func serveFixture(w http.ResponseWriter, iso string) {
	data, err := fixtures.ReadFile("fixtures/" + iso + ".json")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"elements":[]}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
