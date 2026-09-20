// Package share owns the visit share link (ADR-0011): a third, strictly
// visit-scoped read-only credential for relatives. It is a permission to view
// one journey, never an identity — it authenticates on exactly one route and
// nothing else authenticates there. Other modules must depend on its Service,
// never its Repo.
package share

import (
	"context"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrInvalidLink is the single answer for every resolution failure — unknown,
// expired, or revoked token — so the endpoint cannot be probed for which
// links existed (the same posture as a login failure in ADR-0010).
var ErrInvalidLink = apperr.New(apperr.KindUnauthorized, "share link is invalid, expired, or revoked")

// ErrTooManyLinks guards the active-links-per-visit cap (ADR-0011 §4).
var ErrTooManyLinks = apperr.New(apperr.KindConflict, "too many active share links for this visit")

// MaxActiveLinks caps how many still-active links one visit may have; the
// next creation attempt returns ErrTooManyLinks (409). A brake on token churn.
const MaxActiveLinks = 5

// LinkSecret is the one-time response to creating a link: the raw token,
// shown exactly once and stored nowhere (only its sha256 is persisted).
type LinkSecret struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Coarse statuses are all a relative may see (NFR-03): waiting to be served,
// being served right now, or finished.
const (
	StatusWaiting   = "WAITING"
	StatusInService = "IN_SERVICE"
	StatusDone      = "DONE"
)

// SharedStep is the current step as a relative may see it: a human-readable
// Thai title, the coarse status, and where the service point is by display
// name and floor. No keys, no codes, no identifiers of any kind.
type SharedStep struct {
	Title            string `json:"title"`
	Status           string `json:"status"`
	ServicePointName string `json:"servicePointName,omitempty"`
	FloorName        string `json:"floorName,omitempty"`
}

// SharedJourney is the redacted view a share link opens (ADR-0011 §3). It is
// deliberately its own schema — never journey.View — so a future field added
// to the full view cannot silently leak here. What is absent is absent by
// construction: patient name, patientRef, visitId, stepKey, clinicCode,
// orderRefs, and the full step list never enter this struct.
type SharedJourney struct {
	Status      string      `json:"status"`
	CurrentStep *SharedStep `json:"currentStep,omitempty"`
	UpdatedAt   time.Time   `json:"updatedAt"`
	ExpiresAt   time.Time   `json:"expiresAt"`
}

// ShareLink is the persisted row (minus the token, which exists only as
// LinkSecret at creation time).
type ShareLink struct {
	TokenHash string
	VisitID   string
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

// Active reports whether the link can still be resolved.
func (l ShareLink) Active(now time.Time) bool {
	return l.RevokedAt == nil && l.ExpiresAt.After(now)
}

// Repo is the persistence port of this module. GetByTokenHash returns the
// row as stored — including expired and revoked ones — so the caller can fold
// every failure into one indistinguishable answer.
type Repo interface {
	Create(ctx context.Context, link ShareLink) error
	GetByTokenHash(ctx context.Context, tokenHash string) (ShareLink, error)
	CountActive(ctx context.Context, visitID string) (int, error)
	RevokeActive(ctx context.Context, visitID string) (int, error)
}
