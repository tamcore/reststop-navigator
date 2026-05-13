// Package serve runs the Reststop Navigator HTTP server with graceful shutdown.
package serve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/api"
	"github.com/tamcore/reststop-navigator/internal/cache"
	"github.com/tamcore/reststop-navigator/internal/config"
	"github.com/tamcore/reststop-navigator/internal/overpass"
	"github.com/tamcore/reststop-navigator/internal/stops"
)

const (
	readHeaderTime  = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

// Run starts the HTTP server with on-demand tile-based caching and blocks
// until SIGINT/SIGTERM is received.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("serve: load config: %w", err)
	}

	rdbOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("serve: parse redis url: %w", err)
	}
	rdb := redis.NewClient(rdbOpts)
	defer func() { _ = rdb.Close() }()

	overpassClient := overpass.NewClient(cfg.OverpassEndpoints)
	tiles := cache.NewTileCache(rdb, overpassClient)
	stopsSvc := stops.NewService(tiles)

	// Start periodic cache stats reporter (every 5 minutes); cancelled on shutdown.
	statsCtx, statsCancel := context.WithCancel(context.Background())
	defer statsCancel()
	tiles.StartStatsReporter(statsCtx, 5*time.Minute)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           api.NewRouter(stopsSvc),
		ReadHeaderTimeout: readHeaderTime,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-stop:
		slog.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
