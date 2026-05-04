// Package config loads runtime configuration from RESTSTOP_* environment
// variables.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// Config is the validated runtime configuration.
type Config struct {
	ListenAddr        string
	RedisURL          string
	OverpassEndpoints []string
	RefreshInterval   time.Duration
}

// Load reads configuration from RESTSTOP_* env vars, applying defaults for
// unset/empty values. Returns an error if any value is malformed (e.g. a bad
// duration).
func Load() (Config, error) {
	cfg := Config{
		ListenAddr:        envOrDefault("RESTSTOP_LISTEN_ADDR", ":8080"),
		RedisURL:          envOrDefault("RESTSTOP_REDIS_URL", "redis://localhost:6379/0"),
		OverpassEndpoints: parseCSV(os.Getenv("RESTSTOP_OVERPASS_ENDPOINTS"), overpass.DefaultEndpoints),
		RefreshInterval:   7 * 24 * time.Hour,
	}

	if raw := os.Getenv("RESTSTOP_REFRESH_INTERVAL"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return Config{}, fmt.Errorf("config: RESTSTOP_REFRESH_INTERVAL %q: %w", raw, err)
		}
		cfg.RefreshInterval = d
	}

	return cfg, nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseCSV(raw string, def []string) []string {
	if raw == "" {
		out := make([]string, len(def))
		copy(out, def)
		return out
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
