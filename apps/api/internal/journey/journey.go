// Package journey owns the CarePath-side journey projection: visits and
// ordered service steps derived from canonical HIS events. Per ADR-0008 the
// HIS remains the system of record for visit and step status; this module
// stores only derived state and depends solely on the canonical his types —
// never on a vendor or mock HIS shape.
package journey

import (
	"context"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrNotFound is returned when no journey is projected for the visit ID.
var ErrNotFound = apperr.New(apperr.KindNotFound, "journey not found")

// Step is one projected service step. ServicePointID is nil when no service
// point is configured for the step's external service code — an explicit
// unmapped state, not an error (#21 AC3).
type Step struct {
	Sequence       int
	ServiceCode    string
	Status         string
	ServicePointID *string
}

// Visit is the projected journey of one HIS visit.
type Visit struct {
	VisitID    string
	PatientRef string
	Status     string
	Steps      []Step
	SyncedAt   time.Time
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; the service wraps its calls in transactions, so
// repository calls always join the ambient transaction carried by ctx.
type Repo interface {
	UpsertVisit(ctx context.Context, visit Visit) error
	GetVisit(ctx context.Context, visitID string) (Visit, error)
	MarkEventApplied(ctx context.Context, eventID, visitID string) error
	EventApplied(ctx context.Context, eventID string) (bool, error)
}
