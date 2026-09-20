package journey

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

func spID(s string) *string { return &s }

func f64(v float64) *float64 { return &v }

func queueVisit() Visit {
	lab := "LAB:1"
	return Visit{
		VisitID: "VISIT-Q", Status: VisitActive,
		Steps: []Step{
			{StepKey: "REGISTRATION", Sequence: 1, Kind: KindRegistration, Status: StepCompleted},
			{StepKey: lab, Sequence: 2, Kind: KindLab, Status: StepReady, ServicePointID: spID("SP-LAB")},
			{StepKey: "CASHIER", Sequence: 3, Kind: KindCashier, Status: StepPending, ServicePointID: spID("SP-CASHIER")},
			// READY but unmapped: no service point, no queue entry.
			{StepKey: "PHARMACY", Sequence: 4, Kind: KindPharmacy, Status: StepPending},
		},
	}
}

// The honest-no-data rule (#101): a service point with no samples today must
// serialize null, never 0 — the client renders "ยังไม่มีข้อมูล" vs "0 นาที"
// from exactly this difference.
func TestShapeQueueNoDataStaysNull(t *testing.T) {
	view := shapeQueue(queueVisit(), map[string]SPQueueStats{"SP-LAB": {WaitingAhead: 2}}, time.Now())

	if len(view.Steps) != 1 {
		t.Fatalf("steps = %d, want 1 (only the mapped READY step)", len(view.Steps))
	}
	q := view.Steps[0]
	if q.StepKey != "LAB:1" || q.ServicePointID != "SP-LAB" {
		t.Fatalf("step = %+v, want LAB:1 at SP-LAB", q)
	}
	if q.WaitingAhead != 2 {
		t.Fatalf("waitingAhead = %d, want 2", q.WaitingAhead)
	}
	if q.AvgWaitMinutes != nil {
		t.Fatalf("avgWaitMinutes = %v, want nil (no samples)", *q.AvgWaitMinutes)
	}
	if q.EstimatedWaitMinutes != nil {
		t.Fatalf("estimatedWaitMinutes = %v, want nil when average is nil", *q.EstimatedWaitMinutes)
	}

	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"avgWaitMinutes":null`) ||
		!strings.Contains(string(raw), `"estimatedWaitMinutes":null`) {
		t.Fatalf("JSON must carry explicit nulls, got %s", raw)
	}
}

func TestShapeQueueEstimateIsProduct(t *testing.T) {
	view := shapeQueue(queueVisit(),
		map[string]SPQueueStats{"SP-LAB": {WaitingAhead: 3, AvgWaitMinutes: f64(12.5)}},
		time.Now())

	q := view.Steps[0]
	if q.AvgWaitMinutes == nil || *q.AvgWaitMinutes != 12.5 {
		t.Fatalf("avgWaitMinutes = %v, want 12.5", q.AvgWaitMinutes)
	}
	if q.EstimatedWaitMinutes == nil || *q.EstimatedWaitMinutes != 37.5 {
		t.Fatalf("estimatedWaitMinutes = %v, want 37.5 (3 × 12.5)", q.EstimatedWaitMinutes)
	}
}

// Two actionable steps at the same service point (§6 same-phase unordered)
// each get their own entry carrying the same stats — the patient sees the
// queue for whichever they act on.
func TestShapeQueueTwoStepsSamePoint(t *testing.T) {
	visit := queueVisit()
	visit.Steps = append(visit.Steps, Step{
		StepKey: "XRAY:1", Sequence: 5, Kind: KindXray, Status: StepReady, ServicePointID: spID("SP-LAB"),
	})
	stats := map[string]SPQueueStats{"SP-LAB": {WaitingAhead: 1, AvgWaitMinutes: f64(4)}}

	view := shapeQueue(visit, stats, time.Now())

	if len(view.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(view.Steps))
	}
	for _, q := range view.Steps {
		if q.WaitingAhead != 1 || q.AvgWaitMinutes == nil || *q.AvgWaitMinutes != 4 {
			t.Fatalf("step %s stats = %+v, want the shared SP-LAB picture", q.StepKey, q)
		}
	}
}

// A finished visit has no queue question to answer: empty list, not an error
// and not the stale plan's steps.
func TestShapeQueueFinalVisitIsEmpty(t *testing.T) {
	for _, status := range []string{VisitCompleted, VisitCancelled} {
		visit := queueVisit()
		visit.Status = status
		view := shapeQueue(visit, map[string]SPQueueStats{"SP-LAB": {WaitingAhead: 9}}, time.Now())
		if len(view.Steps) != 0 || view.Steps == nil {
			t.Fatalf("status %s: steps = %#v, want empty non-nil", status, view.Steps)
		}
	}
}

// GetQueue asks the repository for exactly the bound service points of the
// actionable steps, self-excluded, and passes the visit through shaping.
func TestGetQueueService(t *testing.T) {
	repo := newFakeRepo()
	visit := queueVisit()
	repo.visits["VISIT-Q"] = visit
	repo.queueStats = map[string]SPQueueStats{"SP-LAB": {WaitingAhead: 4, AvgWaitMinutes: f64(10)}}
	svc := newTestService(&fakeHIS{}, repo)

	view, err := svc.GetQueue(context.Background(), "VISIT-Q")
	if err != nil {
		t.Fatalf("GetQueue: %v", err)
	}
	if repo.queueExcludeVisitID != "VISIT-Q" {
		t.Fatalf("QueueStats exclude = %q, want the asking visit itself", repo.queueExcludeVisitID)
	}
	if len(repo.queueSPIDs) != 1 || repo.queueSPIDs[0] != "SP-LAB" {
		t.Fatalf("QueueStats spIDs = %v, want [SP-LAB] (READY+mapped only)", repo.queueSPIDs)
	}
	if repo.queueTZ != "Asia/Bangkok" {
		t.Fatalf("QueueStats tz = %q, want the service's configured tz", repo.queueTZ)
	}
	if len(view.Steps) != 1 || view.Steps[0].WaitingAhead != 4 {
		t.Fatalf("view = %+v, want the LAB step with 4 ahead", view.Steps)
	}
}

// A visit with nothing actionable must not query the repository at all —
// there is no service point to ask about.
func TestGetQueueNoActionableSteps(t *testing.T) {
	repo := newFakeRepo()
	visit := queueVisit()
	visit.Steps[1].Status = StepCompleted // LAB done → nothing READY
	repo.visits["VISIT-Q"] = visit
	svc := newTestService(&fakeHIS{}, repo)

	view, err := svc.GetQueue(context.Background(), "VISIT-Q")
	if err != nil {
		t.Fatalf("GetQueue: %v", err)
	}
	if repo.queueSPIDs != nil {
		t.Fatalf("QueueStats must not run without actionable steps, asked for %v", repo.queueSPIDs)
	}
	if len(view.Steps) != 0 {
		t.Fatalf("steps = %d, want 0", len(view.Steps))
	}
}

func TestGetQueueUnknownVisit(t *testing.T) {
	svc := newTestService(&fakeHIS{}, newFakeRepo())
	_, err := svc.GetQueue(context.Background(), "VISIT-NONE")
	if apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("err = %v, want journey.ErrNotFound", err)
	}
}
