package visitlink

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"log/slog"
	"time"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/identity"
)

// Service is the only entry point other modules may call.
type Service interface {
	// Mint returns the visit's patient link, creating it on first call.
	// Repeat mints return the same URL (a re-print must not silently kill
	// the slip already in the patient's hand); rotate discards the current
	// token — a new token is minted and the old one stops redeeming.
	Mint(ctx context.Context, visitID string, rotate bool) (Link, error)
	// Redeem exchanges a presented link token for the visit it addresses,
	// minting the #96 claim for the identity. Every failure — unknown,
	// rotated, cancelled, past-grace, or minted under an older secret — is
	// the same 404.
	Redeem(ctx context.Context, id identity.Identity, token string) (visitID string, err error)
}

type service struct {
	repo    Repo
	claims  Claims
	secret  []byte
	baseURL string
	grace   time.Duration
	log     *slog.Logger
}

func NewService(repo Repo, claims Claims, secret []byte, patientBaseURL string, completedGrace time.Duration, log *slog.Logger) Service {
	return &service{repo: repo, claims: claims, secret: secret, baseURL: patientBaseURL,
		grace: completedGrace, log: log}
}

// token computes the credential for a visit at a token version:
// HMAC-SHA256 over (visit id, version), truncated to 16 bytes (128-bit) —
// unguessable at URL scale, short enough to keep the printed QR sparse.
// Deterministic on purpose: a repeat mint recomputes the same URL without
// the raw token ever being stored, and rotating VISIT_LINK_SECRET makes
// every previously printed link stop redeeming (see Redeem) — the operator's
// break-glass for a suspected leak.
func (s *service) token(visitID string, version int) string {
	mac := hmac.New(sha256.New, s.secret)
	fmt.Fprintf(mac, "visit-link:%s:%d", visitID, version)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)[:16])
}

// tokenHash is the only form persisted or looked up — the raw token is
// never stored, matching the share-link rule (ADR-0011 §5).
func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func (s *service) Mint(ctx context.Context, visitID string, rotate bool) (Link, error) {
	hash, version, exists, err := s.repo.Current(ctx, visitID)
	if err != nil {
		return Link{}, err
	}
	if !exists {
		return Link{}, ErrNotFound
	}
	if hash != "" && !rotate {
		current := s.token(visitID, version)
		if tokenHash(current) == hash {
			return s.link(visitID, current), nil
		}
		// The stored hash no longer matches the deterministic token — the
		// secret changed under us. Fall through and mint a fresh version;
		// Redeem's HMAC check has already killed links under the old secret.
		s.log.Warn("visit link no longer reproducible; minting a new version", "visit_id", visitID)
	}
	version++
	raw := s.token(visitID, version)
	if err := s.repo.Store(ctx, visitID, tokenHash(raw), version); err != nil {
		return Link{}, err
	}
	return s.link(visitID, raw), nil
}

func (s *service) link(visitID, raw string) Link {
	// Fragment, not query: the token must never reach an access log
	// (ADR-0011 §5 precedent). The web stashes it before LIFF redirects.
	return Link{VisitID: visitID, VisitURL: s.baseURL + "/patient/journey#vt=" + raw}
}

func (s *service) Redeem(ctx context.Context, id identity.Identity, raw string) (string, error) {
	if raw == "" || len(raw) != base64.RawURLEncoding.EncodedLen(16) {
		return "", ErrNotFound
	}
	presented, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(presented) != 16 {
		return "", ErrNotFound
	}
	visitID, status, version, completedAt, err := s.repo.Resolve(ctx, tokenHash(raw))
	if err != nil {
		return "", err
	}
	// The hash index found the row; now prove the token is the one this
	// secret mints — a constant-time comparison over the base64 form, and
	// the reason a rotated VISIT_LINK_SECRET revokes every printed link at
	// once.
	if !hmac.Equal([]byte(raw), []byte(s.token(visitID, version))) {
		return "", ErrNotFound
	}
	if !s.linkLive(status, completedAt, time.Now()) {
		return "", ErrNotFound
	}
	if err := s.claims.Claim(ctx, id, visitID); err != nil {
		return "", err
	}
	return visitID, nil
}

// linkLive is the lazy lifetime policy (#136): ACTIVE lives, CANCELLED dies
// at once, COMPLETED lives out its grace from the projection's completed_at
// so the patient can still open the summary while walking out. A COMPLETED
// row with no completed_at (projected before 000022) never had a live token
// — deny rather than guess. The grace is evaluated at redeem time, so a
// VISIT_LINK_COMPLETED_GRACE change applies without touching the event path.
func (s *service) linkLive(status string, completedAt *time.Time, now time.Time) bool {
	switch status {
	case his.VisitActive:
		return true
	case his.VisitCompleted:
		return completedAt != nil && now.Sub(*completedAt) <= s.grace
	default:
		return false
	}
}
