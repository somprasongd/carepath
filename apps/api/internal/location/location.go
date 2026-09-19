// Package location defines the provider port that resolves a patient's
// current location from QR codes, Zigbee positioning, manual selection, or
// any future source (ADR-0004). Providers translate their own protocol onto
// the canonical Observation; journey and navigation code consume only that
// normalized result and never see provider protocol details.
package location

import (
	"context"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// Source identifies which provider produced an observation.
type Source string

const (
	SourceQR     Source = "QR"     // MVP baseline: a scanned CarePath QR code
	SourceZigbee Source = "ZIGBEE" // phase 2: positioning service over Zigbee tags
	SourceManual Source = "MANUAL" // fallback/debug: a human-picked node
)

// ErrUnknownSource is returned when no provider is registered for a source.
var ErrUnknownSource = apperr.New(apperr.KindInvalid, "unknown location source")

// ErrInvalidFix is returned when a provider's raw input cannot be resolved
// (empty payload, unparseable message).
var ErrInvalidFix = apperr.New(apperr.KindInvalid, "invalid location fix")

// ErrUnknownNode is returned when a fix resolves to a node that is not in
// the navigation graph and so cannot be a canonical location.
var ErrUnknownNode = apperr.New(apperr.KindInvalid, "location fix resolves to an unknown navigation node")

// ErrNoLocation is returned when a visit has no recorded observation yet.
var ErrNoLocation = apperr.New(apperr.KindNotFound, "no location recorded for visit")

// Observation is the canonical current-location result (the
// LocationObservation of the domain model): whatever a provider measures —
// a scanned QR code, a Zigbee zone fix, a manually picked point — lands on
// this shape. NodeID is the globally unique navigation node id
// ("<floorId>/<localId>"); Zone and Confidence are optional (a QR or manual
// fix is exact, a Zigbee fix is not); ObservedAt carries the provider's
// measurement time and is stamped by the service when the provider has none.
type Observation struct {
	VisitID    string    `json:"visitId"`
	NodeID     string    `json:"nodeId"`
	FloorID    string    `json:"floorId"`
	Zone       *string   `json:"zone,omitempty"`
	Source     Source    `json:"source"`
	Confidence *float64  `json:"confidence,omitempty"`
	ObservedAt time.Time `json:"observedAt"`
}

// Validate reports whether the observation carries the fields every
// canonical fix must have, independent of its source.
func (o Observation) Validate() error {
	switch {
	case o.Source == "":
		return apperr.New(apperr.KindInvalid, "observation missing source")
	case o.NodeID == "":
		return apperr.New(apperr.KindInvalid, "observation missing nodeId")
	case o.Confidence != nil && (*o.Confidence < 0 || *o.Confidence > 1):
		return apperr.New(apperr.KindInvalid, "observation confidence must be within [0,1]")
	}
	return nil
}

// Provider resolves one provider-specific raw fix into a canonical
// Observation. Raw is opaque outside the implementation — the QR provider
// parses scanned payloads, the Zigbee provider positioning-service messages,
// the manual provider a node id — which is what keeps protocol knowledge
// out of the rest of the system (ADR-0004). The service validates and
// canonicalizes the returned observation against the navigation graph, so
// implementations only need to translate position: at least NodeID, plus
// whatever metadata (zone, confidence, observed-at) their protocol carries.
type Provider interface {
	// Source names the source this provider serves.
	Source() Source
	// Resolve translates one raw fix into a canonical observation.
	Resolve(ctx context.Context, raw string) (Observation, error)
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; callers get a tx-bound Querier via the ctx passed
// to Service methods, so repository calls always join the ambient
// transaction.
type Repo interface {
	// Record appends one observation as the visit's latest location.
	Record(ctx context.Context, obs Observation) error
	// Latest returns the most recently recorded observation of a visit.
	Latest(ctx context.Context, visitID string) (Observation, error)
}
