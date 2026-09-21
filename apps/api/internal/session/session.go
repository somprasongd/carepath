// Package session owns the CarePath-issued session that stands in for a
// verified external identity (LINE LIFF today; a demo bypass for local dev).
// Other modules must depend on its Service, never its Repo.
package session

import (
	"context"
	"time"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
)

// ErrNotFound covers a missing or expired session token: from the caller's
// side this is "not authenticated", not "resource missing".
var ErrNotFound = apperr.New(apperr.KindUnauthorized, "session not found or expired")

// ErrDemoAuthDisabled is returned for a demo-source request when the backend
// gate (ALLOW_DEMO_AUTH) is off.
var ErrDemoAuthDisabled = apperr.New(apperr.KindUnauthorized, "demo auth is disabled")

// ErrLineAuthNotConfigured is returned for a "line" source request when the
// server was started without LINE_CHANNEL_ID — a deployment/config problem,
// not a bad request from the caller.
var ErrLineAuthNotConfigured = apperr.New(apperr.KindInternal, "line auth is not configured")

// ErrVisitTokenNotConfigured is returned for a "visit-token" source request
// when the visit-link env (HIS_API_KEY + VISIT_LINK_SECRET) is unset: the
// session bootstrap fails closed exactly like the mint routes it depends on.
var ErrVisitTokenNotConfigured = apperr.New(apperr.KindUnauthorized, "visit-token sessions are not configured")

// Session is a CarePath-issued credential standing in for a verified identity.
// VisitID is ephemeral: only the visit-token bootstrap sets it (the visit the
// slip token addressed, echoed once in the create response). It is not
// persisted — the claim table is the durable identity→visit binding.
type Session struct {
	Token     string
	Identity  identity.Identity
	VisitID   string
	ExpiresAt time.Time
}

// ErrVisitNotFound is the claim-side counterpart of journey.ErrNotFound:
// claiming a visit that has never been projected answers with the same
// status and text, so a caller cannot distinguish "unknown visit" from any
// other journey 404.
var ErrVisitNotFound = apperr.New(apperr.KindNotFound, "journey not found")

// ErrNotClaimed is returned by RequirePatientVisit when the session's
// identity has no claim on the visit it addressed. It carries the exact
// status and text of journey.ErrNotFound on purpose: "this visit exists but
// is not yours" and "this visit does not exist" must be indistinguishable
// from the outside (#96).
var ErrNotClaimed = apperr.New(apperr.KindNotFound, "journey not found")

// Repo is the persistence port of this module.
type Repo interface {
	Create(ctx context.Context, s Session) error
	// Get returns ErrNotFound for a token with no row, or one whose row has
	// already expired — callers never need to check ExpiresAt themselves.
	Get(ctx context.Context, token string) (Session, error)
}

// ClaimRepo persists which identity may read which visit (#96). Claims are
// keyed by identity, not token: a re-login mints a fresh session for the
// same LINE user and keeps every claim.
type ClaimRepo interface {
	// Claim records the identity's ownership of the visit. Idempotent for a
	// repeat claim; ErrVisitNotFound when no such visit is projected.
	Claim(ctx context.Context, id identity.Identity, visitID string) error
	// HasClaimed reports whether the identity may read the visit.
	HasClaimed(ctx context.Context, id identity.Identity, visitID string) (bool, error)
}

// VisitTokenResolver validates a slip-held visit-link token (#136) and names
// the visit it addresses, without recording a claim — the session bootstrap
// (source "visit-token") resolves first, then claims for the visit-scoped
// identity it mints. The port lives here because session cannot import
// visitlink (visitlink's routes sit behind session.RequireSession); main.go
// injects the visitlink service, keeping the dependency one-way. A nil
// resolver disables the source — fail closed, like the mint itself.
type VisitTokenResolver interface {
	ResolveToken(ctx context.Context, raw string) (visitID string, err error)
}
