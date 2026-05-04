// Package serve runs the Reststop Navigator HTTP server with graceful shutdown.
package serve

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tamcore/reststop-navigator/internal/api"
)

const (
	defaultAddr     = ":8080"
	readHeaderTime  = 10 * time.Second
	shutdownTimeout = 10 * time.Second
)

// Run starts the HTTP server and blocks until SIGINT/SIGTERM is received.
func Run() error {
	addr := os.Getenv("RESTSTOP_LISTEN_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           api.NewRouter(),
		ReadHeaderTimeout: readHeaderTime,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", addr)
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

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	return srv.Shutdown(ctx)
}
