// Package visitlink owns the slip-held patient visit link (#136, FR-18
// layer 2): the credential a printed navigation slip carries, minted by
// CarePath for the HIS and redeemed into a patient_visit_claim (#96). The
// HIS visit id is guessable by design — this token is the part that must
// not be.
package visitlink

import (
	"context"
	"time"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
)

// ErrNotFound is this module's one 404, carrying the journey read's exact
// text: an unknown visit, an unknown token, a rotated token, and a token
// past its completed-grace are all indistinguishable from the outside.
var ErrNotFound = apperr.New(apperr.KindNotFound, "journey not found")

// ErrBadKey is the HIS-side mint surface's 401: a missing or wrong
// X-HIS-API-Key. It exists only on the mint endpoint — claim-by-token
// authenticates with a patient session instead.
var ErrBadKey = apperr.New(apperr.KindUnauthorized, "missing or invalid HIS API key")

// Repo is the persistence port of this module.
type Repo interface {
	// Current returns the visit's stored token hash and version. A visit
	// that was never projected reports exists=false — mint answers the
	// journey-shaped 404, same as an unprojected visit read.
	Current(ctx context.Context, visitID string) (hash string, version int, exists bool, err error)
	// Store upserts the token row; a rotate replaces the hash and version,
	// which is what kills the previous token (its hash is gone).
	Store(ctx context.Context, visitID, tokenHash string, version int) error
	// Resolve maps a presented token's hash to its visit, token version,
	// and the projection state the lifetime policy needs. No row →
	// ErrNotFound.
	Resolve(ctx context.Context, tokenHash string) (visitID, status string, version int, completedAt *time.Time, err error)
}

// Claims is the #96 claim port: redeeming a link token mints the same
// identity→visit ownership row typing a VN used to.
type Claims interface {
	Claim(ctx context.Context, id identity.Identity, visitID string) error
}

// Link is the mint result the HIS prints.
type Link struct {
	VisitID  string `json:"visitId"`
	VisitURL string `json:"visitUrl"`
}
