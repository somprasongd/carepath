package journey

import (
	"time"

	"carepath/apps/api/internal/servicepoint"
)

// Canonical statuses of the projection (contract enums; vendor codes are
// translated inside the HIS adapter per ADR-0008 §6).
const (
	statusVisitCompleted = "COMPLETED"
	statusStepStarted    = "STARTED"
	statusStepReady      = "READY"
)

// StepView is one projected step resolved to its service point for display.
// A nil ServicePoint is the explicit unmapped state (#21 AC3).
type StepView struct {
	Sequence       int                        `json:"sequence"`
	ServiceCode    string                     `json:"serviceCode"`
	Status         string                     `json:"status"`
	ServicePointID *string                    `json:"servicePointId"`
	ServicePoint   *servicepoint.ServicePoint `json:"servicePoint,omitempty"`
}

// View is the patient-facing journey: the projection with steps ordered by
// sequence and the deterministic current/next resolution. Completed makes a
// finished visit unambiguous without parsing status (#18 AC4).
type View struct {
	VisitID    string     `json:"visitId"`
	PatientRef string     `json:"patientRef"`
	Status     string     `json:"status"`
	Completed  bool       `json:"completed"`
	Steps      []StepView `json:"steps"`
	Current    *StepView  `json:"current,omitempty"`
	Next       *StepView  `json:"next,omitempty"`
	SyncedAt   time.Time  `json:"syncedAt"`
}
