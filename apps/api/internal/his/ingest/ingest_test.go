package ingest

import (
	"context"
	"testing"
	"time"

	"log/slog"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
)

type fakeSource struct {
	pages     []his.EventPage
	calls     int
	lastAfter string
	lastLimit int
	err       error
}

func (f *fakeSource) Events(_ context.Context, after string, limit int) (his.EventPage, error) {
	if f.err != nil {
		return his.EventPage{}, f.err
	}
	f.calls++
	if f.calls == 1 {
		f.lastAfter = after
	}
	f.lastLimit = limit
	if f.calls > len(f.pages) {
		return his.EventPage{}, nil // drained
	}
	return f.pages[f.calls-1], nil
}

type fakeApplier struct {
	applied []string
	failOn  map[string]error
}

func (f *fakeApplier) ApplyHISEvent(_ context.Context, event his.Event) error {
	if err, ok := f.failOn[event.EventID]; ok {
		return err
	}
	f.applied = append(f.applied, event.EventID)
	return nil
}

type fakeCheckpoint struct {
	last    string
	saved   []string
	failAll bool
}

func (f *fakeCheckpoint) LastEventID(context.Context) (string, error) {
	return f.last, nil
}

func (f *fakeCheckpoint) Save(_ context.Context, lastEventID string) error {
	if f.failAll {
		return apperr.New(apperr.KindInternal, "checkpoint unavailable")
	}
	f.last = lastEventID
	f.saved = append(f.saved, lastEventID)
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(&discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError}))
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

func page(after string, ids ...string) his.EventPage {
	events := make([]his.Event, 0, len(ids))
	for _, id := range ids {
		events = append(events, his.Event{EventID: id, VisitID: "VISIT-001", Type: his.EventVisitUpdated})
	}
	return his.EventPage{Events: events, NextAfter: after}
}

func TestPollOnceDrainsFeedAndSavesCursorPerPage(t *testing.T) {
	source := &fakeSource{pages: []his.EventPage{
		page("EVT-000002", "EVT-000001", "EVT-000002"),
		page("EVT-000003", "EVT-000003"),
	}}
	applier := &fakeApplier{}
	checkpoint := &fakeCheckpoint{}

	consumed, err := New(source, applier, checkpoint, testLogger()).PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	if consumed != 3 {
		t.Fatalf("consumed = %d, want 3", consumed)
	}
	if len(applier.applied) != 3 || applier.applied[2] != "EVT-000003" {
		t.Fatalf("applied = %v, want all three events in order", applier.applied)
	}
	if len(checkpoint.saved) != 2 || checkpoint.last != "EVT-000003" {
		t.Fatalf("saves = %v (last %q), want a save per page ending at EVT-000003", checkpoint.saved, checkpoint.last)
	}
	if source.lastLimit != pageSize {
		t.Fatalf("limit = %d, want %d", source.lastLimit, pageSize)
	}
}

func TestPollOnceResumesFromCursor(t *testing.T) {
	source := &fakeSource{pages: []his.EventPage{page("EVT-000004", "EVT-000004")}}
	checkpoint := &fakeCheckpoint{last: "EVT-000003"}

	if _, err := New(source, &fakeApplier{}, checkpoint, testLogger()).PollOnce(context.Background()); err != nil {
		t.Fatalf("PollOnce: %v", err)
	}
	if source.lastAfter != "EVT-000003" {
		t.Fatalf("first request after = %q, want the stored cursor EVT-000003", source.lastAfter)
	}
}

// The cursor is only saved after a page is fully applied, so a mid-page
// failure leaves it before the failing event and the next poll retries it.
func TestPollOnceRetriesFromFailedEvent(t *testing.T) {
	source := &fakeSource{pages: []his.EventPage{
		page("EVT-000002", "EVT-000001", "EVT-000002"),
	}}
	applier := &fakeApplier{failOn: map[string]error{
		"EVT-000002": apperr.New(apperr.KindUpstream, "HIS unavailable"),
	}}
	checkpoint := &fakeCheckpoint{}

	if _, err := New(source, applier, checkpoint, testLogger()).PollOnce(context.Background()); err == nil {
		t.Fatal("PollOnce should surface the apply failure")
	}
	if len(checkpoint.saved) != 0 {
		t.Fatalf("saves = %v, want none (the failing page was never completed)", checkpoint.saved)
	}
	if len(applier.applied) != 1 {
		t.Fatalf("applied = %v, want only EVT-000001", applier.applied)
	}

	// Recover: the retry re-delivers EVT-000002 from the stored cursor.
	applier.failOn = nil
	source.calls = 0
	if _, err := New(source, applier, checkpoint, testLogger()).PollOnce(context.Background()); err != nil {
		t.Fatalf("retry PollOnce: %v", err)
	}
	if applier.applied[len(applier.applied)-1] != "EVT-000002" {
		t.Fatalf("applied = %v, want the retried event applied", applier.applied)
	}
}

// A malformed envelope must not block the feed: it is skipped and the cursor
// advances past it.
func TestPollOnceSkipsInvalidEvents(t *testing.T) {
	bad := his.Event{EventID: "EVT-BAD", Type: his.EventVisitUpdated} // no visitId
	source := &fakeSource{pages: []his.EventPage{{Events: []his.Event{bad}, NextAfter: "EVT-000003"}}}
	applier := &fakeApplier{failOn: map[string]error{
		"EVT-BAD": apperr.New(apperr.KindInvalid, "event envelope missing visitId"),
	}}
	checkpoint := &fakeCheckpoint{}

	consumed, err := New(source, applier, checkpoint, testLogger()).PollOnce(context.Background())
	if err != nil {
		t.Fatalf("PollOnce: %v (invalid events must be skipped)", err)
	}
	if consumed != 0 {
		t.Fatalf("consumed = %d, want 0", consumed)
	}
	if checkpoint.last != "EVT-000003" {
		t.Fatalf("cursor = %q, want it advanced past the invalid event", checkpoint.last)
	}
}

func TestPollOnceCheckpointFailurePropagates(t *testing.T) {
	source := &fakeSource{pages: []his.EventPage{page("EVT-000001", "EVT-000001")}}
	checkpoint := &fakeCheckpoint{failAll: true}

	if _, err := New(source, &fakeApplier{}, checkpoint, testLogger()).PollOnce(context.Background()); err == nil {
		t.Fatal("PollOnce should surface checkpoint failures")
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	source := &fakeSource{}
	poller := New(source, &fakeApplier{}, &fakeCheckpoint{}, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { poller.Run(ctx, 10*time.Millisecond); close(done) }()

	deadline := time.After(2 * time.Second)
	cancel()
	for {
		select {
		case <-done:
			return
		case <-deadline:
			t.Fatal("Run did not stop after ctx cancellation")
		}
	}
}
