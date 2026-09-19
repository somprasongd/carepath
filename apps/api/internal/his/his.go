// Package his defines the outbound port to the Hospital Information System.
// All HIS access in CarePath goes through this interface (ADR-0005); the
// implementation can be swapped from Mock HIS to a real HIS adapter without
// touching the modules that consume visits.
package his

import (
	"context"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// Canonical event types of the integration contract (mock-his.yaml schema
// EventType). A real HIS adapter translates vendor event names onto these.
const (
	EventVisitOpened      = "visit.opened"
	EventVisitUpdated     = "visit.updated"
	EventServiceRequested = "service.requested"
	EventServiceStarted   = "service.started"
	EventServiceCompleted = "service.completed"
	EventServiceCancelled = "service.cancelled"
)

// ErrVisitNotFound is returned when the HIS reports no visit for the given ID.
var ErrVisitNotFound = apperr.New(apperr.KindNotFound, "visit not found")

// Canonical command targets (mock-his.yaml TransitionCommand schema). A real
// HIS adapter translates its own action vocabulary onto these.
const (
	CommandToStarted   = "STARTED"
	CommandToCompleted = "COMPLETED"
	CommandToCancelled = "CANCELLED"
)

// TransitionCommand is the canonical CarePath→HIS command changing one
// service step's status (ADR-0008 §1). CommandID is caller-assigned and the
// HIS-side idempotency key: replaying it is a no-op that returns the current
// step.
type TransitionCommand struct {
	CommandID string `json:"commandId"`
	To        string `json:"to"`
}

// ErrUpstream marks any HIS-side failure (network error, unexpected status,
// malformed payload) and maps to a 502 upstream response.
var ErrUpstream = apperr.New(apperr.KindUpstream, "upstream HIS error")

// VisitStep mirrors the HIS visit step contract
// (packages/contracts/openapi/mock-his.yaml).
type VisitStep struct {
	Sequence    int    `json:"sequence"`
	ServiceCode string `json:"serviceCode"`
	Status      string `json:"status"`
}

// Visit is a visit as reported by the HIS.
type Visit struct {
	VisitID    string      `json:"visitId"`
	PatientRef string      `json:"patientRef"`
	Status     string      `json:"status"`
	Steps      []VisitStep `json:"steps"`
}

// Event is the canonical HIS event envelope (ADR-0008): transport-agnostic —
// today delivered by the pull feed, later possibly by webhook or message.
// EventID is HIS-assigned and the idempotency key for consumers.
type Event struct {
	EventID    string         `json:"eventId"`
	OccurredAt time.Time      `json:"occurredAt"`
	VisitID    string         `json:"visitId"`
	PatientRef string         `json:"patientRef"`
	Type       string         `json:"type"`
	Payload    map[string]any `json:"payload"`
}

// Validate reports whether the envelope carries the fields every canonical
// event must have, independent of its type-specific payload.
func (e Event) Validate() error {
	switch {
	case e.EventID == "":
		return apperr.New(apperr.KindInvalid, "event envelope missing eventId")
	case e.VisitID == "":
		return apperr.New(apperr.KindInvalid, "event envelope missing visitId")
	case e.Type == "":
		return apperr.New(apperr.KindInvalid, "event envelope missing type")
	}
	return nil
}

// EventPage is one page of the append-only event feed, oldest first.
// NextAfter is the cursor for the next page; empty when there are no more
// events.
type EventPage struct {
	Events    []Event `json:"events"`
	NextAfter string  `json:"nextAfter"`
}

// Client is the HIS port. The HIS is an external system, so implementations
// must not participate in CarePath database transactions.
type Client interface {
	GetVisit(ctx context.Context, visitID string) (Visit, error)
	Events(ctx context.Context, after string, limit int) (EventPage, error)
	// TransitionStep forwards one step-status command. The HIS owns
	// transition legality: it reports unknown visit/step (NotFound), illegal
	// or out-of-order targets (Conflict), and unknown targets (Invalid).
	TransitionStep(ctx context.Context, visitID string, sequence int, cmd TransitionCommand) (VisitStep, error)
}
