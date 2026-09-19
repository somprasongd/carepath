// Package his defines the outbound port to the Hospital Information System.
// All HIS access in CarePath goes through this interface (ADR-0005); the
// implementation can be swapped from Mock HIS to a real HIS adapter without
// touching the modules that consume visits.
package his

import (
	"context"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrVisitNotFound is returned when the HIS reports no visit for the given ID.
var ErrVisitNotFound = apperr.New(apperr.KindNotFound, "visit not found")

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

// Client is the HIS port. The HIS is an external system, so implementations
// must not participate in CarePath database transactions.
type Client interface {
	GetVisit(ctx context.Context, visitID string) (Visit, error)
}
