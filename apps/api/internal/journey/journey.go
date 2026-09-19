// Package journey owns the CarePath-side journey plan: a visit and its
// ordered steps, derived from canonical HIS facts (ADR-0009). The HIS is the
// system of record for the facts (visit, clinic assignment, orders,
// encounter completion); CarePath is the system of record for the plan
// itself — which steps exist, their order, and their status. This module
// depends solely on the canonical his types — never on a vendor or mock HIS
// shape.
package journey

import (
	"context"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrNotFound is returned when no journey is projected for the visit ID.
var ErrNotFound = apperr.New(apperr.KindNotFound, "journey not found")

// ErrNoOpenRound is returned when a staff close-round request names a clinic
// the visit has no open round at.
var ErrNoOpenRound = apperr.New(apperr.KindNotFound, "visit has no open round at this clinic")

// Step kinds (contract schema StepKind, ADR-0009).
const (
	KindRegistration = "REGISTRATION"
	KindClinic       = "CLINIC"
	KindLab          = "LAB"
	KindXray         = "XRAY"
	KindEKG          = "EKG"
	KindUltrasound   = "ULTRASOUND"
	KindCashier      = "CASHIER"
	KindPharmacy     = "PHARMACY"
)

// Canonical step statuses (contract enum, ADR-0009 §5 adds WAITING).
const (
	StepPending   = "PENDING"
	StepWaiting   = "WAITING"
	StepReady     = "READY"
	StepStarted   = "STARTED"
	StepCompleted = "COMPLETED"
	StepCancelled = "CANCELLED"
)

// Canonical visit statuses.
const (
	VisitActive    = "ACTIVE"
	VisitCompleted = "COMPLETED"
	VisitCancelled = "CANCELLED"
)

// Staff-writable transition targets (contract TransitionRequest.to). Per
// ADR-0009 this is a local CarePath write — it is never forwarded to the HIS.
const (
	CommandToStarted   = "STARTED"
	CommandToCompleted = "COMPLETED"
	CommandToCancelled = "CANCELLED"
)

// TransitionCommand is a staff command changing one step's status.
// CommandID is caller-assigned: replaying it is a no-op that re-audits but
// does not re-apply an already-applied change.
type TransitionCommand struct {
	CommandID string
	To        string
}

// Step is one step of the journey plan CarePath derived from HIS facts.
// StepKey is the stable identity across replans; Sequence is display order
// only. ClinicCode/Round are set when Kind is CLINIC; OrderRefs names the HIS
// orders a diagnostic step represents. ServicePointID is nil when no service
// point is configured for this step's binding key — an explicit unmapped
// state, not an error.
type Step struct {
	StepKey        string
	Sequence       int
	Kind           string
	ClinicCode     *string
	Round          *int
	OrderRefs      []string
	Status         string
	ServicePointID *string
}

// Visit is the projected journey of one HIS visit.
type Visit struct {
	VisitID     string
	PatientRef  string
	PatientName string
	Status      string
	Steps       []Step
	SyncedAt    time.Time
}

// CommandAudit is the CarePath-side operational record of one staff
// transition command: timestamp and source at minimum.
type CommandAudit struct {
	CommandID string
	VisitID   string
	StepKey   string
	ToStatus  string
	Source    string
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; the service wraps its calls in transactions, so
// repository calls always join the ambient transaction carried by ctx.
type Repo interface {
	UpsertVisit(ctx context.Context, visit Visit) error
	GetVisit(ctx context.Context, visitID string) (Visit, error)
	// ListVisits returns every projected visit with its steps, freshest sync
	// first — ordering is part of the port's contract (#37).
	ListVisits(ctx context.Context) ([]Visit, error)
	MarkEventApplied(ctx context.Context, eventID, visitID string) error
	EventApplied(ctx context.Context, eventID string) (bool, error)
	InsertCommandAudit(ctx context.Context, audit CommandAudit) error
	// CloseRound records a clinic round as explicitly confirmed finished
	// (ADR-0009 §4), whether by the HIS's encounter.completed fact or the
	// staff override. Idempotent.
	CloseRound(ctx context.Context, visitID, stepKey string) error
	// ClosedRounds returns every stepKey closed for this visit.
	ClosedRounds(ctx context.Context, visitID string) (map[string]bool, error)
}
