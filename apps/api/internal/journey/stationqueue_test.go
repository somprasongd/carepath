package journey

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func stationAt(offset time.Duration) *time.Time {
	t := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC).Add(offset)
	return &t
}

func stationEntry(visitID, status string, readyAt, startedAt *time.Time) StationQueueEntry {
	return StationQueueEntry{
		VisitID: visitID, StepKey: "ORDERTYPE:LAB:1", Kind: KindLab,
		Status: status, PatientRef: "PAT-" + visitID, PatientName: "ผู้ป่วย " + visitID,
		Sequence: 2, TotalSteps: 4,
		ReadyAt: readyAt, StartedAt: startedAt,
	}
}

// The queue's core promise: waiting order is arrival order (#102 —
// "เรียงตามเวลาที่เข้ามารอ"), regardless of the order rows come back in.
func TestShapeStationQueueWaitingOrdersByArrival(t *testing.T) {
	entries := []StationQueueEntry{
		stationEntry("VISIT-LATE", StepReady, stationAt(20*time.Minute), nil),
		stationEntry("VISIT-EARLY", StepReady, stationAt(5*time.Minute), nil),
		stationEntry("VISIT-UNKNOWN", StepReady, nil, nil),
		stationEntry("VISIT-SERVED", StepStarted, stationAt(1*time.Minute), stationAt(15*time.Minute)),
	}

	queue := shapeStationQueue("SP-LAB", entries, time.Now().UTC())

	if len(queue.Serving) != 1 || queue.Serving[0].VisitID != "VISIT-SERVED" {
		t.Fatalf("serving = %+v, want only VISIT-SERVED", queue.Serving)
	}
	wantOrder := []string{"VISIT-EARLY", "VISIT-LATE", "VISIT-UNKNOWN"}
	for i, want := range wantOrder {
		if queue.Waiting[i].VisitID != want {
			t.Fatalf("waiting[%d] = %s, want %s (waiting = %+v)", i, queue.Waiting[i].VisitID, want, queue.Waiting)
		}
	}
}

// Empty and non-empty lists alike must serialize as arrays, never null —
// the client renders an empty state from the same shape.
func TestShapeStationQueueEmptyListsAreArrays(t *testing.T) {
	queue := shapeStationQueue("SP-EMPTY", nil, time.Now().UTC())

	raw, err := json.Marshal(queue)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"serving":[]`) || !strings.Contains(string(raw), `"waiting":[]`) {
		t.Fatalf("empty queue = %s, want empty arrays for serving and waiting", raw)
	}
}

// Serving orders by call time when a point runs more than one counter.
func TestShapeStationQueueServingOrdersLatestCallFirst(t *testing.T) {
	entries := []StationQueueEntry{
		stationEntry("VISIT-CALLED-EARLIER", StepStarted, stationAt(1*time.Minute), stationAt(12*time.Minute)),
		stationEntry("VISIT-CALLED-JUST-NOW", StepStarted, stationAt(10*time.Minute), stationAt(30*time.Minute)),
	}

	queue := shapeStationQueue("SP-LAB", entries, time.Now().UTC())

	if queue.Serving[0].VisitID != "VISIT-CALLED-JUST-NOW" || queue.Serving[1].VisitID != "VISIT-CALLED-EARLIER" {
		t.Fatalf("serving order = %s, %s; want the latest call leading",
			queue.Serving[0].VisitID, queue.Serving[1].VisitID)
	}
}

func TestGetStationQueueService(t *testing.T) {
	repo := newFakeRepo()
	repo.stationQueue = []StationQueueEntry{
		stationEntry("VISIT-A", StepReady, stationAt(2*time.Minute), nil),
		stationEntry("VISIT-B", StepReady, stationAt(1*time.Minute), nil),
	}
	svc := newTestService(nil, repo)

	queue, err := svc.GetStationQueue(context.Background(), "SP-LAB")
	if err != nil {
		t.Fatalf("GetStationQueue: %v", err)
	}
	if repo.stationSPID != "SP-LAB" {
		t.Fatalf("repo saw servicePointId %q, want SP-LAB", repo.stationSPID)
	}
	if len(queue.Waiting) != 2 || queue.Waiting[0].VisitID != "VISIT-B" {
		t.Fatalf("waiting = %+v, want VISIT-B (earlier arrival) first", queue.Waiting)
	}
}
