package analytics

import (
	"context"
	"testing"
	"time"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

// fakeRepo returns canned data; every test controls exactly what the
// database would have aggregated, so the assertions target the shaping
// rules (null-not-zero, bottleneck selection) in isolation.
type fakeRepo struct {
	data Data
	err  error
}

func (f *fakeRepo) Fetch(ctx context.Context, tz string) (Data, error) {
	return f.data, f.err
}

// fakeTx runs fn immediately, like a real transaction would for a read.
type fakeTx struct{}

func (fakeTx) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

var _ db.Transactor = fakeTx{}

func mustService(t *testing.T) Service {
	t.Helper()
	svc, err := NewService(&fakeRepo{}, fakeTx{}, "Asia/Bangkok")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func fp(v float64) *float64 { return &v }

func TestOverviewRejectsUnsupportedWindow(t *testing.T) {
	svc := mustService(t)
	for _, window := range []string{"week", "yesterday", "", "Today"} {
		_, err := svc.Overview(context.Background(), window)
		if apperr.KindOf(err) != apperr.KindInvalid {
			t.Fatalf("window %q: kind = %v, want invalid", window, apperr.KindOf(err))
		}
	}
}

func TestOverviewAcceptsToday(t *testing.T) {
	svc, err := NewService(&fakeRepo{data: Data{AsOf: time.Now()}}, fakeTx{}, "UTC")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	overview, err := svc.Overview(context.Background(), "today")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if overview.Window != "today" {
		t.Fatalf("window = %q, want today", overview.Window)
	}
}

func TestNewServiceRejectsUnknownTimezone(t *testing.T) {
	if _, err := NewService(&fakeRepo{}, fakeTx{}, "Mars/Olympus"); err == nil {
		t.Fatal("unknown timezone must fail at construction")
	}
}

// Averages over no samples are null in the database and must stay null in
// the response — a 0 would read as "everyone was served instantly".
func TestOverviewKeepsNullDistinctFromZero(t *testing.T) {
	repo := &fakeRepo{data: Data{
		AsOf:         time.Now(),
		ActiveVisits: 3,
		Points: []PointData{{
			ServicePointID: "SP-A", Code: "A", Name: "Point A",
			WaitingNow: 2, InProgressNow: 0, CompletedCount: 0,
		}},
	}}
	svc, err := NewService(repo, fakeTx{}, "Asia/Bangkok")
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	overview, err := svc.Overview(context.Background(), "today")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if overview.Summary.AvgWaitMinutes != nil {
		t.Errorf("summary avgWait = %v, want nil", *overview.Summary.AvgWaitMinutes)
	}
	if overview.Summary.AvgVisitMinutes != nil {
		t.Errorf("summary avgVisit = %v, want nil", *overview.Summary.AvgVisitMinutes)
	}
	if overview.Summary.BottleneckServicePointID != nil {
		t.Errorf("bottleneck = %v, want nil when no service point has wait data", *overview.Summary.BottleneckServicePointID)
	}
	p := overview.ServicePoints[0]
	if p.AvgWaitMinutes != nil || p.AvgServiceMinutes != nil || p.LongestWaitingMinutes != nil {
		t.Errorf("per-point averages = %v/%v/%v, want all nil", p.AvgWaitMinutes, p.AvgServiceMinutes, p.LongestWaitingMinutes)
	}
	if p.WaitingNow != 2 || p.CompletedCount != 0 {
		t.Errorf("counts = %d/%d, want 2 waiting, 0 completed", p.WaitingNow, p.CompletedCount)
	}
}

func TestOverviewBottleneckIsHighestAvgWait(t *testing.T) {
	repo := &fakeRepo{data: Data{
		Points: []PointData{
			{ServicePointID: "SP-A", Code: "A", AvgWaitMinutes: fp(5), WaitingNow: 1},
			{ServicePointID: "SP-B", Code: "B", AvgWaitMinutes: fp(12.5), WaitingNow: 0},
			{ServicePointID: "SP-C", Code: "C", AvgWaitMinutes: fp(9), WaitingNow: 3},
		},
	}}
	svc, _ := NewService(repo, fakeTx{}, "Asia/Bangkok")
	overview, err := svc.Overview(context.Background(), "today")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if got := overview.Summary.BottleneckServicePointID; got == nil || *got != "SP-B" {
		t.Fatalf("bottleneck = %v, want SP-B (highest average)", got)
	}
}

// Equal averages break by who is still waiting now — the queue the manager
// can actually act on.
func TestOverviewBottleneckTieBreaksByWaitingNow(t *testing.T) {
	repo := &fakeRepo{data: Data{
		Points: []PointData{
			{ServicePointID: "SP-A", Code: "A", AvgWaitMinutes: fp(10), WaitingNow: 4},
			{ServicePointID: "SP-B", Code: "B", AvgWaitMinutes: fp(10), WaitingNow: 7},
			{ServicePointID: "SP-C", Code: "C", AvgWaitMinutes: fp(10), WaitingNow: 1},
		},
	}}
	svc, _ := NewService(repo, fakeTx{}, "Asia/Bangkok")
	overview, err := svc.Overview(context.Background(), "today")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if got := overview.Summary.BottleneckServicePointID; got == nil || *got != "SP-B" {
		t.Fatalf("bottleneck = %v, want SP-B (tie, more waiting now)", got)
	}
}

// A service point with waitingNow but no average yet must not outrank one
// with data: nil averages are skipped, never treated as +inf or 0.
func TestOverviewBottleneckSkipsNoDataServicePoints(t *testing.T) {
	repo := &fakeRepo{data: Data{
		Points: []PointData{
			{ServicePointID: "SP-A", Code: "A", AvgWaitMinutes: nil, WaitingNow: 9},
			{ServicePointID: "SP-B", Code: "B", AvgWaitMinutes: fp(3), WaitingNow: 0},
		},
	}}
	svc, _ := NewService(repo, fakeTx{}, "Asia/Bangkok")
	overview, err := svc.Overview(context.Background(), "today")
	if err != nil {
		t.Fatalf("Overview: %v", err)
	}
	if got := overview.Summary.BottleneckServicePointID; got == nil || *got != "SP-B" {
		t.Fatalf("bottleneck = %v, want SP-B", got)
	}
}
