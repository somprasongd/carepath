package journey

import (
	"time"
)

// QueueStep is one actionable step's queue picture (FR-17): how many people
// are waiting ahead of this patient at the step's service point right now,
// and how long a wait that point has averaged so far today.
//
// WaitingAhead counts other visits' READY steps at the service point — the
// patient's own READY step is excluded, so 0 means "you are next". Averages
// over zero samples stay nil all the way to the JSON (#101, following the
// analytics null-not-zero contract): "no data yet" and "zero minutes" are
// different facts, and a guessed number would send the patient walking.
//
// EstimatedWaitMinutes is the naive product WaitingAhead × AvgWaitMinutes —
// a rank-order estimate, not a promise; nil whenever AvgWaitMinutes is nil.
type QueueStep struct {
	StepKey              string   `json:"stepKey"`
	ServicePointID       string   `json:"servicePointId"`
	WaitingAhead         int      `json:"waitingAhead"`
	AvgWaitMinutes       *float64 `json:"avgWaitMinutes"`
	EstimatedWaitMinutes *float64 `json:"estimatedWaitMinutes"`
}

// QueueView is the wire shape of GET /api/v1/journeys/{visitId}/queue: one
// entry per currently-READY step that is bound to a service point. A visit
// with nothing actionable (or every actionable step unmapped) answers an
// empty list, not an error — the queue question just has no referent.
type QueueView struct {
	VisitID string      `json:"visitId"`
	AsOf    time.Time   `json:"asOf"`
	Steps   []QueueStep `json:"steps"`
}

// SPQueueStats is the repository's per-service-point contribution: the count
// of READY steps from other visits, and today's average experienced wait
// (latest READY → latest STARTED per step, the same pairing analytics uses)
// as nil when the window holds no samples.
type SPQueueStats struct {
	WaitingAhead   int
	AvgWaitMinutes *float64
}

// shapeQueue builds the patient-facing queue view from the visit's plan and
// the per-service-point stats. Pure: the null-not-zero contract, the
// self-exclusion bookkeeping's display side, and the estimate product are
// all decidable without a database (#101).
func shapeQueue(visit Visit, stats map[string]SPQueueStats, now time.Time) QueueView {
	view := QueueView{VisitID: visit.VisitID, AsOf: now, Steps: []QueueStep{}}
	if visit.Status == VisitCompleted || visit.Status == VisitCancelled {
		return view
	}
	for _, step := range visit.Steps {
		if step.Status != StepReady || step.ServicePointID == nil {
			continue
		}
		s := stats[*step.ServicePointID] // zero value: 0 ahead, nil average
		q := QueueStep{
			StepKey: step.StepKey, ServicePointID: *step.ServicePointID,
			WaitingAhead: s.WaitingAhead, AvgWaitMinutes: s.AvgWaitMinutes,
		}
		if s.AvgWaitMinutes != nil {
			estimate := float64(s.WaitingAhead) * *s.AvgWaitMinutes
			q.EstimatedWaitMinutes = &estimate
		}
		view.Steps = append(view.Steps, q)
	}
	return view
}
