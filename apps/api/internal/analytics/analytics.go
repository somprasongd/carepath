// Package analytics serves the executive insights dashboard (#86, epic #83)
// from the append-only step timeline the journey module writes (#85) —
// "journey เขียน · analytics อ่าน". It is read-only over
// carepath.journey_step_status_event (plus current state in journey_step and
// the service_point catalog), aggregates in SQL, and never exposes
// patient-level data (NFR-03): counts, averages, and one bottleneck id.
package analytics

import (
	"context"
	"time"
)

// WindowToday is the only aggregation window the MVP serves; the service
// rejects anything else as invalid rather than guessing.
const WindowToday = "today"

// Service is the only entry point other modules may call.
type Service interface {
	// Overview returns the executive overview for one window. Averages over
	// zero samples come back as nil, not 0 — "no data yet" and "zero
	// minutes" are different facts, and the dashboard renders them
	// differently (#87).
	Overview(ctx context.Context, window string) (Overview, error)
}

// Overview is the wire shape of GET /api/v1/analytics/overview.
type Overview struct {
	AsOf          time.Time             `json:"asOf"`
	Window        string                `json:"window"`
	Summary       Summary               `json:"summary"`
	ServicePoints []ServicePointMetrics `json:"servicePoints"`
}

// Summary holds the visit-wide numbers.
type Summary struct {
	ActiveVisits             int      `json:"activeVisits"`
	AvgWaitMinutes           *float64 `json:"avgWaitMinutes"`
	AvgVisitMinutes          *float64 `json:"avgVisitMinutes"`
	BottleneckServicePointID *string  `json:"bottleneckServicePointId"`
}

// ServicePointMetrics holds one service point's row in the overview.
type ServicePointMetrics struct {
	ServicePointID        string   `json:"servicePointId"`
	Code                  string   `json:"code"`
	Name                  string   `json:"name"`
	WaitingNow            int      `json:"waitingNow"`
	InProgressNow         int      `json:"inProgressNow"`
	LongestWaitingMinutes *float64 `json:"longestWaitingMinutes"`
	AvgWaitMinutes        *float64 `json:"avgWaitMinutes"`
	AvgServiceMinutes     *float64 `json:"avgServiceMinutes"`
	CompletedCount        int      `json:"completedCount"`
}

// Data is everything one Fetch lifts out of the database for the "today"
// window. The service shapes it into an Overview — bottleneck selection and
// the null-not-zero contract are pure Go, so they are unit-testable without
// a database.
type Data struct {
	AsOf            time.Time
	ActiveVisits    int
	AvgWaitMinutes  *float64
	AvgVisitMinutes *float64
	Points          []PointData
}

// PointData is the database's per-service-point contribution.
type PointData struct {
	ServicePointID        string
	Code                  string
	Name                  string
	WaitingNow            int
	InProgressNow         int
	LongestWaitingMinutes *float64
	AvgWaitMinutes        *float64
	AvgServiceMinutes     *float64
	CompletedCount        int
}

// Repo is the persistence port. Fetch computes every metric for the local
// day containing now() in tz — midnight boundaries are resolved by the
// database via AT TIME ZONE, never from the Go process's clock or locale.
type Repo interface {
	Fetch(ctx context.Context, tz string) (Data, error)
}
