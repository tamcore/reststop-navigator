package config_test

import (
	"testing"
	"time"

	"github.com/tamcore/reststop-navigator/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("RESTSTOP_LISTEN_ADDR", "")
	t.Setenv("RESTSTOP_REDIS_URL", "")
	t.Setenv("RESTSTOP_OVERPASS_ENDPOINTS", "")
	t.Setenv("RESTSTOP_REFRESH_INTERVAL", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q", cfg.ListenAddr)
	}
	if cfg.RedisURL != "redis://localhost:6379/0" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if len(cfg.OverpassEndpoints) == 0 {
		t.Error("OverpassEndpoints should default to non-empty list")
	}
	if cfg.RefreshInterval != 7*24*time.Hour {
		t.Errorf("RefreshInterval = %v", cfg.RefreshInterval)
	}
}

func TestLoad_FromEnv(t *testing.T) {
	t.Setenv("RESTSTOP_LISTEN_ADDR", ":9090")
	t.Setenv("RESTSTOP_REDIS_URL", "redis://redis.example.com:6380/3")
	t.Setenv("RESTSTOP_OVERPASS_ENDPOINTS", "https://op1.example,https://op2.example")
	t.Setenv("RESTSTOP_REFRESH_INTERVAL", "1h")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ListenAddr != ":9090" {
		t.Errorf("ListenAddr = %q", cfg.ListenAddr)
	}
	if cfg.RedisURL != "redis://redis.example.com:6380/3" {
		t.Errorf("RedisURL = %q", cfg.RedisURL)
	}
	if len(cfg.OverpassEndpoints) != 2 || cfg.OverpassEndpoints[0] != "https://op1.example" {
		t.Errorf("OverpassEndpoints = %v", cfg.OverpassEndpoints)
	}
	if cfg.RefreshInterval != time.Hour {
		t.Errorf("RefreshInterval = %v", cfg.RefreshInterval)
	}
}

func TestLoad_AdminPasswordDefaultsEmpty(t *testing.T) {
	t.Setenv("RESTSTOP_ADMIN_PASSWORD", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AdminPassword != "" {
		t.Errorf("AdminPassword = %q, want empty", cfg.AdminPassword)
	}
}

func TestLoad_AdminPasswordFromEnv(t *testing.T) {
	t.Setenv("RESTSTOP_ADMIN_PASSWORD", "hunter2")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AdminPassword != "hunter2" {
		t.Errorf("AdminPassword = %q, want %q", cfg.AdminPassword, "hunter2")
	}
}

func TestLoad_RejectsBadDuration(t *testing.T) {
	t.Setenv("RESTSTOP_REFRESH_INTERVAL", "not-a-duration")
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error from invalid duration")
	}
}

func TestLoad_TrimsWhitespaceInEndpointsCSV(t *testing.T) {
	t.Setenv("RESTSTOP_OVERPASS_ENDPOINTS", " https://a.example , https://b.example ")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OverpassEndpoints[0] != "https://a.example" || cfg.OverpassEndpoints[1] != "https://b.example" {
		t.Errorf("OverpassEndpoints = %v", cfg.OverpassEndpoints)
	}
}
