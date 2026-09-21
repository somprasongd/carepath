package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
)

// Service is the only entry point other modules may call.
type Service interface {
	// Create verifies the presented credential and mints a new session for
	// it: a LINE ID token (source "line"), the demo bypass (source "demo",
	// gated by allowDemo), or a slip-held visit-link token (source
	// "visit-token") — the QR-only front door for hospitals without a LINE
	// OA, which also claims the visit it addresses (ADR-0014).
	Create(ctx context.Context, source, idToken string) (Session, error)
	// Get resolves a bearer token to the identity behind it.
	Get(ctx context.Context, token string) (identity.Identity, error)
}

type service struct {
	repo         Repo
	lineVerifier identity.Verifier
	visitTokens  VisitTokenResolver
	claims       ClaimRepo
	allowDemo    bool
	ttl          time.Duration
}

func NewService(repo Repo, lineVerifier identity.Verifier, visitTokens VisitTokenResolver, claims ClaimRepo, allowDemo bool, ttl time.Duration) Service {
	return &service{repo: repo, lineVerifier: lineVerifier, visitTokens: visitTokens,
		claims: claims, allowDemo: allowDemo, ttl: ttl}
}

func (s *service) Create(ctx context.Context, source, idToken string) (Session, error) {
	id, visitID, err := s.resolveIdentity(ctx, source, idToken)
	if err != nil {
		return Session{}, err
	}

	if visitID != "" {
		// Claim before persisting the session: a failure here (visit not yet
		// projected) costs the scanner a retry, not an orphaned session that
		// can never claim — the visit identity has no other route to a claim
		// once claim-by-name retires with the mint.
		if err := s.claims.Claim(ctx, id, visitID); err != nil {
			return Session{}, err
		}
	}

	token, err := newToken()
	if err != nil {
		return Session{}, apperr.Wrapf(apperr.KindInternal, err, "session: generate token")
	}

	sess := Session{Token: token, Identity: id, VisitID: visitID, ExpiresAt: time.Now().Add(s.ttl)}
	if err := s.repo.Create(ctx, sess); err != nil {
		return Session{}, err
	}
	return sess, nil
}

// resolveIdentity verifies the source's credential. The second return is the
// visit a visit-token addressed ("" for every other source): Create turns it
// into the session's claim, so a slip token alone carries a patient all the
// way in.
func (s *service) resolveIdentity(ctx context.Context, source, idToken string) (identity.Identity, string, error) {
	switch source {
	case "line":
		if s.lineVerifier == nil {
			return identity.Identity{}, "", ErrLineAuthNotConfigured
		}
		id, err := s.lineVerifier.Verify(ctx, idToken)
		return id, "", err
	case "demo":
		if !s.allowDemo {
			return identity.Identity{}, "", ErrDemoAuthDisabled
		}
		return identity.Identity{Source: "demo", ExternalID: "demo-user", DisplayName: "Demo User"}, "", nil
	case "visit-token":
		if s.visitTokens == nil {
			return identity.Identity{}, "", ErrVisitTokenNotConfigured
		}
		visitID, err := s.visitTokens.ResolveToken(ctx, idToken)
		if err != nil {
			return identity.Identity{}, "", err
		}
		// No DisplayName on purpose: patient-facing text is client-owned
		// (ADR-0012) and a visit-scoped identity has nothing to show.
		return identity.Identity{Source: "visit", ExternalID: "visit:" + visitID}, visitID, nil
	default:
		return identity.Identity{}, "", identity.ErrInvalidToken
	}
}

func (s *service) Get(ctx context.Context, token string) (identity.Identity, error) {
	sess, err := s.repo.Get(ctx, token)
	if err != nil {
		return identity.Identity{}, err
	}
	return sess.Identity, nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
