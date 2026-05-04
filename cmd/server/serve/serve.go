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
	"sync"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/api"
	"github.com/tamcore/reststop-navigator/internal/cache"
	"github.com/tamcore/reststop-navigator/internal/config"
	"github.com/tamcore/reststop-navigator/internal/overpass"
)

const (
	readHeaderTime  = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

// Run starts the HTTP server, kicks off the country-dataset hydrator, and
// blocks until SIGINT/SIGTERM is received.
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

	c := cache.NewRedis(rdb)
	overpassClient := overpass.NewClient(cfg.OverpassEndpoints)
	hydrator := cache.NewHydrator(overpassClient, c)

	rootCtx, cancelRoot := context.WithCancel(context.Background())
	defer cancelRoot()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		hydrator.Run(rootCtx, cfg.RefreshInterval)
	}()

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           api.NewRouter(),
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
		cancelRoot()
		wg.Wait()
		return err
	case sig := <-stop:
		slog.Info("shutdown signal received", "signal", sig.String())
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	cancelRoot() // tell the hydrator to stop
	if err := srv.Shutdown(shutdownCtx); err != nil {
		wg.Wait()
		return err
	}
	wg.Wait()
	return nil
}
