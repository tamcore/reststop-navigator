package presence_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/tamcore/reststop-navigator/internal/presence"
)

const testClientID = "b3a4c1d2-5e6f-4a7b-8c9d-0e1f2a3b4c5d"

func newTracker(t *testing.T) (*presence.Tracker, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return presence.NewTracker(rdb), mr
}

func samplePosition() presence.Position {
	return presence.Position{
		Lat:      48.1,
		Lon:      11.5,
		Heading:  92.5,
		Speed:    104,
		Accuracy: 8,
		LastSeen: time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC),
	}
}

func TestRecordAndList_RoundTrip(t *testing.T) {
	t.Parallel()
	tr, _ := newTracker(t)
	ctx := context.Background()

	if err := tr.Record(ctx, testClientID, samplePosition()); err != nil {
		t.Fatalf("Record: %v", err)
	}

	clients, err := tr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("List returned %d clients, want 1", len(clients))
	}
	got := clients[0]
	if got.ClientID != testClientID {
		t.Errorf("ClientID = %q", got.ClientID)
	}
	if got.Lat != 48.1 || got.Lon != 11.5 || got.Heading != 92.5 || got.Speed != 104 || got.Accuracy != 8 {
		t.Errorf("position fields = %+v", got)
	}
	if !got.LastSeen.Equal(time.Date(2026, 6, 12, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("LastSeen = %v", got.LastSeen)
	}
}

func TestRecord_RejectsInvalidClientID(t *testing.T) {
	t.Parallel()
	tr, _ := newTracker(t)
	ctx := context.Background()

	for _, id := range []string{"", "not-a-uuid", "../../etc/passwd", "b3a4c1d2-5e6f-4a7b-8c9d-0e1f2a3b4c5d-extra"} {
		if err := tr.Record(ctx, id, samplePosition()); err == nil {
			t.Errorf("Record(%q) accepted invalid client id", id)
		}
	}

	clients, err := tr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(clients) != 0 {
		t.Fatalf("List returned %d clients, want 0", len(clients))
	}
}

func TestRecord_EntriesExpire(t *testing.T) {
	t.Parallel()
	tr, mr := newTracker(t)
	ctx := context.Background()

	if err := tr.Record(ctx, testClientID, samplePosition()); err != nil {
		t.Fatalf("Record: %v", err)
	}
	mr.FastForward(16 * time.Minute)

	clients, err := tr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(clients) != 0 {
		t.Fatalf("expected entry to expire after TTL, got %d clients", len(clients))
	}
}

func TestList_SkipsCorruptEntries(t *testing.T) {
	t.Parallel()
	tr, mr := newTracker(t)
	ctx := context.Background()

	if err := tr.Record(ctx, testClientID, samplePosition()); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := mr.Set("reststops:presence:v1:00000000-0000-4000-8000-000000000000", "{corrupt"); err != nil {
		t.Fatalf("seed corrupt entry: %v", err)
	}

	clients, err := tr.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("List returned %d clients, want 1 (corrupt entry skipped)", len(clients))
	}
}

func TestIsValidClientID(t *testing.T) {
	t.Parallel()
	valid := []string{testClientID, "00000000-0000-0000-0000-000000000000", "B3A4C1D2-5E6F-4A7B-8C9D-0E1F2A3B4C5D"}
	for _, id := range valid {
		if !presence.IsValidClientID(id) {
			t.Errorf("IsValidClientID(%q) = false, want true", id)
		}
	}
	invalid := []string{"", "short", "b3a4c1d2-5e6f-4a7b-8c9d", "b3a4c1d25e6f4a7b8c9d0e1f2a3b4c5d", "zzzzzzzz-zzzz-zzzz-zzzz-zzzzzzzzzzzz"}
	for _, id := range invalid {
		if presence.IsValidClientID(id) {
			t.Errorf("IsValidClientID(%q) = true, want false", id)
		}
	}
}
