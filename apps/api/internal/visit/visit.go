// Package visit serves the normalized visit view: HIS visit data enriched
// with the next actionable step and its service point.
package visit

import (
	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/servicepoint"
)

// ErrNoNextStep is returned when the visit has no step in a READY state.
var ErrNoNextStep = apperr.New(apperr.KindNotFound, "no next actionable step")

// NextStep is the first READY step of a visit, resolved to its service point
// when one is configured for the step's service code.
type NextStep struct {
	Sequence     int                        `json:"sequence"`
	Status       string                     `json:"status"`
	ServicePoint *servicepoint.ServicePoint `json:"servicePoint,omitempty"`
}

// VisitView embeds the raw HIS visit and adds the resolved next step.
type VisitView struct {
	his.Visit
	Next *NextStep `json:"next,omitempty"`
}
