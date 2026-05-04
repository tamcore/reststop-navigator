// Package cache stores and retrieves country datasets in Redis using a
// two-key swap so in-flight readers always see a consistent version.
package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/overpass"
)

// Redis is the cache backed by go-redis.
type Redis struct {
	client *redis.Client
	ttl    time.Duration
}

// Option configures NewRedis.
type Option func(*Redis)

// WithTTL overrides the per-key TTL applied to versioned datasets and the
// current pointer. Default 30 days.
func WithTTL(d time.Duration) Option {
	return func(r *Redis) { r.ttl = d }
}

// NewRedis builds a Redis cache wrapping the given client.
func NewRedis(client *redis.Client, opts ...Option) *Redis {
	r := &Redis{
		client: client,
		ttl:    30 * 24 * time.Hour,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// WriteDataset stores ds and updates the current-version pointer. On
// pointer-write failure the prior version remains visible to readers.
func (r *Redis) WriteDataset(ctx context.Context, ds overpass.Dataset) error {
	if !overpass.IsSupported(ds.Country) {
		return fmt.Errorf("cache: unsupported country %q", ds.Country)
	}
	if ds.Version == "" {
		return errors.New("cache: dataset version is required")
	}

	payload, err := json.Marshal(ds)
	if err != nil {
		return fmt.Errorf("cache: marshal dataset: %w", err)
	}

	versionedKey := versionKey(ds.Country, ds.Version)
	pointerKey := currentKey(ds.Country)

	if err := r.client.Set(ctx, versionedKey, payload, r.ttl).Err(); err != nil {
		return fmt.Errorf("cache: write versioned key: %w", err)
	}
	if err := r.client.Set(ctx, pointerKey, ds.Version, r.ttl).Err(); err != nil {
		return fmt.Errorf("cache: update current pointer: %w", err)
	}
	return nil
}

// ReadDataset returns the current dataset for c, or an error if missing or
// corrupt.
func (r *Redis) ReadDataset(ctx context.Context, c overpass.CountryISO) (overpass.Dataset, error) {
	if !overpass.IsSupported(c) {
		return overpass.Dataset{}, fmt.Errorf("cache: unsupported country %q", c)
	}

	version, err := r.client.Get(ctx, currentKey(c)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return overpass.Dataset{}, fmt.Errorf("cache: no current version for %q", c)
		}
		return overpass.Dataset{}, fmt.Errorf("cache: read current pointer: %w", err)
	}

	payload, err := r.client.Get(ctx, versionKey(c, version)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return overpass.Dataset{}, fmt.Errorf("cache: version %q for %q gone", version, c)
		}
		return overpass.Dataset{}, fmt.Errorf("cache: read versioned key: %w", err)
	}

	var ds overpass.Dataset
	if err := json.Unmarshal(payload, &ds); err != nil {
		return overpass.Dataset{}, fmt.Errorf("cache: decode dataset: %w", err)
	}
	return ds, nil
}

func versionKey(c overpass.CountryISO, version string) string {
	return fmt.Sprintf("reststops:country:%s:v%s", c, version)
}

func currentKey(c overpass.CountryISO) string {
	return fmt.Sprintf("reststops:country:%s:current", c)
}
