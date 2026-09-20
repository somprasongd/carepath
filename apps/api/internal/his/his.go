// Package his defines the outbound port to the Hospital Information System.
// All HIS access in CarePath goes through this interface (ADR-0005); the
// implementation can be swapped from Mock HIS to a real HIS adapter without
// touching the modules that consume visits.
//
// Per ADR-0009, the HIS has no concept of an ordered patient journey: it
// reports visit/clinic/order/encounter facts, and the journey package derives
// the plan from them. This package therefore carries no step-list or
// step-status concept — those are CarePath-owned (see internal/journey).
package his

import (
	"context"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// Canonical event types of the integration contract (mock-his.yaml schema
// EventType). A real HIS adapter translates vendor event names onto these.
const (
	EventVisitOpened        = "visit.opened"
	EventVisitUpdated       = "visit.updated"
	EventVisitClosed        = "visit.closed"
	EventOrderPlaced        = "order.placed"
	EventOrderPerformed     = "order.performed"
	EventOrderResulted      = "order.resulted"
	EventOrderCancelled     = "order.cancelled"
	EventEncounterStarted   = "encounter.started"
	EventEncounterCompleted = "encounter.completed"
)

// Canonical visit types (mock-his.yaml schema VisitType).
const (
	VisitTypeWalkin      = "WALKIN"
	VisitTypeAppointment = "APPOINTMENT"
)

// Canonical visit statuses (mock-his.yaml schema VisitStatus).
const (
	VisitActive    = "ACTIVE"
	VisitCompleted = "COMPLETED"
	VisitCancelled = "CANCELLED"
)

// Canonical order types (mock-his.yaml schema OrderType). DRUG has no
// PERFORMED/RESULTED distinction — the journey planner treats it as done
// once PLACED (ADR-0009).
const (
	OrderTypeLab  = "LAB"
	OrderTypeXray = "XRAY"
	OrderTypeEKG  = "EKG"
	OrderTypeUS   = "US"
	OrderTypeDrug = "DRUG"
)

// Canonical order statuses (mock-his.yaml schema OrderStatus).
const (
	OrderPlaced    = "PLACED"
	OrderPerformed = "PERFORMED"
	OrderResulted  = "RESULTED"
	OrderCancelled = "CANCELLED"
)

// ErrVisitNotFound is returned when the HIS reports no visit for the given ID.
var ErrVisitNotFound = apperr.New(apperr.KindNotFound, "visit not found")

// ErrUpstream marks any HIS-side failure (network error, unexpected status,
// malformed payload) and maps to a 502 upstream response.
var ErrUpstream = apperr.New(apperr.KindUpstream, "upstream HIS error")

// Clinic is a clinic a visit has been assigned to (mock-his.yaml schema
// Clinic).
type Clinic struct {
	Code string `json:"code"`
	Name string `json:"name,omitempty"`
}

// Order mirrors the HIS order contract (mock-his.yaml schema Order): a lab,
// imaging, EKG, or drug order with its lifecycle.
type Order struct {
	OrderRef        string     `json:"orderRef"`
	OrderType       string     `json:"orderType"`
	OrderName       string     `json:"orderName"`
	OrderedByClinic string     `json:"orderedByClinic"`
	OrderedAt       time.Time  `json:"orderedAt"`
	Status          string     `json:"status"`
	PerformedAt     *time.Time `json:"performedAt,omitempty"`
	ResultedAt      *time.Time `json:"resultedAt,omitempty"`
}

// Visit is a visit as reported by the HIS (mock-his.yaml schema Visit):
// clinical facts only — no ordered step list (ADR-0009).
type Visit struct {
	VisitID     string    `json:"visitId"`
	PatientRef  string    `json:"patientRef"`
	PatientName string    `json:"patientName,omitempty"`
	VisitType   string    `json:"visitType"`
	Status      string    `json:"status"`
	Clinics     []Clinic  `json:"clinics"`
	Orders      []Order   `json:"orders"`
	OpenedAt    time.Time `json:"openedAt"`
}

// Event is the canonical HIS fact envelope (ADR-0008, amended by ADR-0009):
// transport-agnostic — today delivered by the pull feed, later possibly by
// webhook or message. EventID is HIS-assigned and the idempotency key for
// consumers.
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
// must not participate in CarePath database transactions. Per ADR-0009 there
// is no command channel back to the HIS: CarePath owns step status itself, so
// the port is read-only.
type Client interface {
	GetVisit(ctx context.Context, visitID string) (Visit, error)
	Events(ctx context.Context, after string, limit int) (EventPage, error)
}
