// Package floorplan turns an uploaded floor-plan SVG into a trusted artifact.
//
// The rule this package exists to enforce: an uploaded plan is never stored
// or served as the bytes a human handed us. Normalize parses the upload,
// checks it against an allowlist, and re-serializes it from the parsed tree,
// so what the patient app inlines is output this package wrote rather than
// input someone supplied. That is what makes it safe for
// apps/web's FloorPlanMap to keep injecting the plan into the DOM — the
// destination highlight and the drawn route need real SVG elements, and an
// <img> cannot give them that.
//
// Re-serializing (rather than stripping bad parts out of the original text)
// is deliberate: a sanitizer that edits the source string has to out-guess
// every parser quirk that turns harmless-looking bytes into markup, while a
// writer that only ever emits allowlisted names from a parsed tree cannot
// emit anything else no matter what the input did.
//
// See ADR-0003 (SVG plans) and the upload conventions in
// packages/floorplans/README.md.
package floorplan

import (
	"context"
	"fmt"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// Upload limits. They sit far above the real plans (the two MVP floors are
// 12 KB and 7 KB) and exist to bound what gets parsed here and inlined into
// a patient's DOM, not to shape how plans are drawn.
const (
	// MaxBytes is the largest upload accepted, before parsing.
	MaxBytes = 512 << 10
	// MaxElements caps the parsed tree.
	MaxElements = 5000
	// MaxStyleBytes caps the total CSS across every <style> element.
	MaxStyleBytes = 64 << 10
)

// SVGNamespace is the only element namespace a plan may use.
const SVGNamespace = "http://www.w3.org/2000/svg"

// ErrTooLarge is returned before parsing when the upload exceeds MaxBytes.
var ErrTooLarge = apperr.New(apperr.KindInvalid, "floor plan: file is larger than the 512 KB limit")

// ErrMalformed is returned when the upload is not well-formed XML.
var ErrMalformed = apperr.New(apperr.KindInvalid, "floor plan: file is not well-formed SVG")

// rejectf builds the client-facing rejection for a plan that parsed but
// broke a rule. The message reaches the admin who uploaded the file (400s
// are not masked, unlike KindInternal), which is the point: they are at the
// keyboard and can fix the file. Callers pass already-truncated names so a
// hostile upload cannot echo itself back at length.
func rejectf(format string, args ...any) error {
	return apperr.New(apperr.KindInvalid, "floor plan: "+fmt.Sprintf(format, args...))
}

// Expect carries what the map model already believes about this floor, so
// the upload is checked against it while a human is still present. An empty
// ViewBox means this floor has no plan yet and the upload defines its
// coordinate space.
type Expect struct {
	// FloorID is the floor being uploaded to; the plan's g[data-floor] must
	// name it.
	FloorID string
	// ViewBox is the floor's established coordinate space ("0 0 1600 900").
	// Every place.x/y and nav_node.x/y is expressed in it, so a plan that
	// disagrees would silently move every pin and route: it is rejected.
	ViewBox string
	// PlaceIDs are the places the map model holds for this floor. A place
	// the plan does not draw is a warning, not a rejection — the web app
	// already degrades such a place to "not routable yet".
	PlaceIDs []string
	// NodeIDs are the floor-local navigation node ids (e.g. node-reception)
	// the graph holds for this floor. Missing ones warn, for the same reason.
	NodeIDs []string
}

// Warning is a mismatch worth telling the admin about that does not make the
// plan unusable. Warnings are reported on upload and kept with the plan so
// the staff console can show what the current plan does not cover.
type Warning struct {
	// Code is the stable machine-readable reason; display text is the
	// client's (ADR-0012).
	Code string `json:"code"`
	// Ref is the place id, node id, or element the warning is about.
	Ref string `json:"ref"`
}

// Warning codes.
const (
	// WarnPlaceMissing: the map model has this place on this floor, but the
	// plan does not draw it.
	WarnPlaceMissing = "PLACE_MISSING"
	// WarnPlaceUnknown: the plan draws a place the map model does not have.
	WarnPlaceUnknown = "PLACE_UNKNOWN"
	// WarnNodeMissing: the navigation graph has this node, but the plan does
	// not draw it.
	WarnNodeMissing = "NODE_MISSING"
)

// Plan is the trusted artifact: the SVG this package serialized, its digest,
// and the coordinate space it declares. SVG is what gets stored and served;
// the uploaded bytes are kept separately for the admin to download back, and
// are never sent to a patient.
type Plan struct {
	SVG      string    `json:"-"`
	SHA256   string    `json:"sha256"`
	ViewBox  string    `json:"viewBox"`
	Warnings []Warning `json:"warnings"`
}

// ErrPlanNotFound is returned when no stored plan matches the lookup.
var ErrPlanNotFound = apperr.New(apperr.KindNotFound, "floor plan not found")

// Stored is a plan as persisted: the artifact plus who put it there. Plans
// are append-only, so a Stored row is never edited — a new upload is a new
// row, and switching or rolling back moves floor.active_plan_id.
type Stored struct {
	PlanID    string    `json:"planId"`
	FloorID   string    `json:"floorId"`
	SHA256    string    `json:"sha256"`
	ViewBox   string    `json:"viewBox"`
	Warnings  []Warning `json:"warnings"`
	CreatedAt time.Time `json:"createdAt"`
	// CreatedBy is nil for the plans seeded from packages/floorplans, which
	// no admin uploaded.
	CreatedBy *string `json:"createdBy,omitempty"`
}

// Repo is the persistence port of this module. Only this package's postgres
// adapter implements it; callers get a tx-bound Querier via the ctx passed to
// Service methods, so repository calls always join the ambient transaction.
type Repo interface {
	// SVG returns the normalized plan bytes for a floor and digest. Both are
	// in the key because one floor's URL must not be able to address another
	// floor's drawing, and because that is what makes the URL safe to cache
	// forever: the content it names can never change.
	SVG(ctx context.Context, floorID, sha256 string) (string, error)
	// Digests maps plan ids to their digests, for building the URLs the
	// floors listing carries.
	Digests(ctx context.Context, planIDs []string) (map[string]string, error)
	// Insert stores a new plan. Re-storing content a floor already has is
	// not an error: plan ids are derived from the digest, so an unchanged
	// re-upload lands on the row that is already there.
	Insert(ctx context.Context, plan Stored, svg, svgRaw string) error
	Get(ctx context.Context, floorID, planID string) (Stored, error)
	// ListByFloor returns the floor's plans, newest first — the history a
	// rollback picks from.
	ListByFloor(ctx context.Context, floorID string) ([]Stored, error)
	// Raw returns the bytes as uploaded. Only an admin ever sees these;
	// patients are served the normalized artifact.
	Raw(ctx context.Context, floorID, planID string) (string, error)
}
