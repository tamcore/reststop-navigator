package cache

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"

	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// Hydrator pulls country datasets from Overpass and writes them to the cache.
type Hydrator struct {
	client *overpass.Client
	cache  *Redis
	now    func() time.Time
}

// NewHydrator builds a Hydrator wiring an Overpass client to a cache.
func NewHydrator(client *overpass.Client, c *Redis) *Hydrator {
	return &Hydrator{
		client: client,
		cache:  c,
		now:    time.Now,
	}
}

// HydrateCountry fetches the country's dataset from Overpass and stores it in
// the cache. On Overpass failure the prior cached version is left untouched
// (two-key swap semantics in the cache layer).
func (h *Hydrator) HydrateCountry(ctx context.Context, c overpass.CountryISO) error {
	q, err := overpass.CountryQuery(c)
	if err != nil {
		return err
	}
	raw, err := h.client.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("hydrate %q: %w", c, err)
	}
	ds, err := overpass.Decode(raw)
	if err != nil {
		return fmt.Errorf("hydrate %q: %w", c, err)
	}
	overpass.EnrichDataset(&ds)
	ds.Country = c
	ds.Version = strconv.FormatInt(h.now().Unix(), 10)

	if err := h.cache.WriteDataset(ctx, ds); err != nil {
		return fmt.Errorf("hydrate %q: %w", c, err)
	}
	return nil
}

// HydrateAll runs HydrateCountry for every supported country in parallel.
// Per-country failures are aggregated; other countries are NOT aborted.
func (h *Hydrator) HydrateAll(ctx context.Context) error {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)
	for _, c := range overpass.SupportedCountries() {
		c := c
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := h.HydrateCountry(ctx, c); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return errors.Join(errs...)
}

// Run hydrates all countries immediately, then on every interval until ctx is
// cancelled. Per-tick failures are logged but do not abort the loop.
func (h *Hydrator) Run(ctx context.Context, interval time.Duration) {
	if err := h.HydrateAll(ctx); err != nil {
		slog.Warn("initial hydrate had failures", "err", err)
	} else {
		slog.Info("initial hydrate complete")
	}

	t := time.NewTicker(interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := h.HydrateAll(ctx); err != nil {
				slog.Warn("scheduled hydrate had failures", "err", err)
			} else {
				slog.Info("scheduled hydrate complete")
			}
		}
	}
}
