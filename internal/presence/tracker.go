// Package presence records live client positions in Redis with a short TTL,
// giving the admin backend a view of who is currently using the app. Client
// IDs are anonymous, browser-generated UUIDs; entries expire automatically so
// nothing is retained beyond the live window.
package presence

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix = "reststops:presence:v1:"
	// defaultTTL is how long a client stays "live" after its last request.
	defaultTTL = 15 * time.Minute
	// scanBatch is the COUNT hint for the Redis SCAN in List.
	scanBatch = 100
)

var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// IsValidClientID reports whether id looks like a canonical UUID. Anything
// else is rejected at the boundary — client IDs end up in Redis keys.
func IsValidClientID(id string) bool {
	return uuidRe.MatchString(id)
}

// Position is one client's last reported GPS fix.
type Position struct {
	Lat      float64   `json:"lat"`
	Lon      float64   `json:"lon"`
	Heading  float64   `json:"heading"`
	Speed    float64   `json:"speed"` // km/h
	Accuracy float64   `json:"accuracy"`
	LastSeen time.Time `json:"last_seen"`
}

// Client is a live client as returned by List.
type Client struct {
	ClientID string `json:"client_id"`
	Position
}

// Tracker reads and writes presence entries in Redis.
type Tracker struct {
	rdb *redis.Client
	ttl time.Duration
}

// TrackerOption configures NewTracker.
type TrackerOption func(*Tracker)

// WithTTL overrides the presence TTL (default 15 minutes).
func WithTTL(d time.Duration) TrackerOption { return func(t *Tracker) { t.ttl = d } }

// NewTracker builds a Tracker.
func NewTracker(rdb *redis.Client, opts ...TrackerOption) *Tracker {
	t := &Tracker{rdb: rdb, ttl: defaultTTL}
	for _, opt := range opts {
		opt(t)
	}
	return t
}

// Record stores p as the latest position for clientID, refreshing the TTL.
func (t *Tracker) Record(ctx context.Context, clientID string, p Position) error {
	if !IsValidClientID(clientID) {
		return fmt.Errorf("presence: invalid client id %q", clientID)
	}
	buf, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("presence: marshal: %w", err)
	}
	if err := t.rdb.Set(ctx, keyPrefix+clientID, buf, t.ttl).Err(); err != nil {
		return fmt.Errorf("presence: redis set: %w", err)
	}
	return nil
}

// List returns all currently live clients. Corrupt entries are skipped with a
// warning so one bad key cannot break the admin view.
func (t *Tracker) List(ctx context.Context) ([]Client, error) {
	clients := []Client{}
	var cursor uint64
	for {
		keys, next, err := t.rdb.Scan(ctx, cursor, keyPrefix+"*", scanBatch).Result()
		if err != nil {
			return nil, fmt.Errorf("presence: redis scan: %w", err)
		}
		for _, key := range keys {
			payload, err := t.rdb.Get(ctx, key).Bytes()
			if err != nil {
				// Key may have expired between SCAN and GET; skip.
				continue
			}
			var p Position
			if err := json.Unmarshal(payload, &p); err != nil {
				slog.Warn("presence: skipping corrupt entry", "key", key, "error", err)
				continue
			}
			clients = append(clients, Client{ClientID: key[len(keyPrefix):], Position: p})
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return clients, nil
}
