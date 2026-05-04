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

// HydrateCountry fetches the country's dataset from Overpass in 4 quadrant
// sub-bboxes and stores the merged result in the cache. On Overpass failure
// for any sub-bbox the entire country hydrate is aborted and the prior cached
// version is left untouched.
func (h *Hydrator) HydrateCountry(ctx context.Context, c overpass.CountryISO) error {
	bboxes, err := overpass.CountryBBoxes(c)
	if err != nil {
		return err
	}

	merged := overpass.Dataset{Country: c}
	for i, bb := range bboxes {
		raw, err := h.client.Query(ctx, overpass.BBoxQuery(bb))
		if err != nil {
			return fmt.Errorf("hydrate %q quadrant %d: %w", c, i, err)
		}
		part, err := overpass.Decode(raw)
		if err != nil {
			return fmt.Errorf("hydrate %q quadrant %d: %w", c, i, err)
		}
		merged.Ways = append(merged.Ways, part.Ways...)
		merged.Stops = append(merged.Stops, part.Stops...)
		merged.Amenities = append(merged.Amenities, part.Amenities...)
	}

	overpass.EnrichDataset(&merged)
	merged.Version = strconv.FormatInt(h.now().Unix(), 10)

	if err := h.cache.WriteDataset(ctx, merged); err != nil {
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
