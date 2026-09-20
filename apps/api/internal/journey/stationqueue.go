package journey

import (
	"sort"
	"time"
)

// StationQueueEntry is one visit's unfinished step at a service point, as
// the station's queue console sees it (#102, FR-15). PatientRef/PatientName
// are patient-level data and ride this staff surface only — never the
// analytics module (NFR-03). Sequence/TotalSteps let the client say where
// the step sits in the visit ("ขั้นที่ 2 จาก 4") without a second read.
type StationQueueEntry struct {
	VisitID     string  `json:"visitId"`
	StepKey     string  `json:"stepKey"`
	Kind        string  `json:"kind"`
	ClinicCode  *string `json:"clinicCode"`
	Round       *int    `json:"round"`
	Sequence    int     `json:"sequence"`
	TotalSteps  int     `json:"totalSteps"`
	Status      string  `json:"status"` // READY (waiting) or STARTED (being served)
	PatientRef  string  `json:"patientRef"`
	PatientName string  `json:"patientName"`
	// ReadyAt is when the step entered READY — the moment the patient
	// joined this queue. Nil only when no timeline row survived (a row
	// seeded outside the normal flow); such entries still queue, last.
	ReadyAt *time.Time `json:"readyAt"`
	// StartedAt is set while the step is being served.
	StartedAt *time.Time `json:"startedAt"`
}

// StationQueue is the working picture of one service point: who is being
// served and who is waiting, longest-waiting first. Serving and Waiting are
// separate lists because they answer different questions ("who is at the
// counter" vs "who is next"); each entry is a full StationQueueEntry so the
// same transition controls work on either.
type StationQueue struct {
	ServicePointID string              `json:"servicePointId"`
	AsOf           time.Time           `json:"asOf"`
	Serving        []StationQueueEntry `json:"serving"`
	Waiting        []StationQueueEntry `json:"waiting"`
}

// shapeStationQueue splits and orders raw entries into the station view.
// Waiting orders by arrival (ReadyAt asc, nil last, then visit id for a
// stable tie-break); serving orders latest call first — the patient just
// called is the one at the counter. Both lists are non-nil so JSON carries
// [] rather than null.
func shapeStationQueue(servicePointID string, entries []StationQueueEntry, now time.Time) StationQueue {
	queue := StationQueue{
		ServicePointID: servicePointID,
		AsOf:           now,
		Serving:        []StationQueueEntry{},
		Waiting:        []StationQueueEntry{},
	}
	for _, entry := range entries {
		switch entry.Status {
		case StepStarted:
			queue.Serving = append(queue.Serving, entry)
		case StepReady:
			queue.Waiting = append(queue.Waiting, entry)
		}
	}
	sort.SliceStable(queue.Waiting, func(i, j int) bool {
		a, b := queue.Waiting[i], queue.Waiting[j]
		if (a.ReadyAt == nil) != (b.ReadyAt == nil) {
			return a.ReadyAt != nil // known arrivals before unknown
		}
		if a.ReadyAt != nil && b.ReadyAt != nil && !a.ReadyAt.Equal(*b.ReadyAt) {
			return a.ReadyAt.Before(*b.ReadyAt)
		}
		return a.VisitID < b.VisitID
	})
	sort.SliceStable(queue.Serving, func(i, j int) bool {
		a, b := queue.Serving[i], queue.Serving[j]
		if (a.StartedAt == nil) != (b.StartedAt == nil) {
			return a.StartedAt != nil
		}
		if a.StartedAt != nil && b.StartedAt != nil && !a.StartedAt.Equal(*b.StartedAt) {
			return a.StartedAt.After(*b.StartedAt) // latest call first
		}
		return a.VisitID < b.VisitID
	})
	return queue
}
