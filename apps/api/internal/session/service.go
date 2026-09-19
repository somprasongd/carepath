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
	// Create verifies idToken (source "line") or the demo bypass (source
	// "demo", gated by allowDemo) and mints a new session for it.
	Create(ctx context.Context, source, idToken string) (Session, error)
	// Get resolves a bearer token to the identity behind it.
	Get(ctx context.Context, token string) (identity.Identity, error)
}

type service struct {
	repo         Repo
	lineVerifier identity.Verifier
	allowDemo    bool
	ttl          time.Duration
}

func NewService(repo Repo, lineVerifier identity.Verifier, allowDemo bool, ttl time.Duration) Service {
	return &service{repo: repo, lineVerifier: lineVerifier, allowDemo: allowDemo, ttl: ttl}
}

func (s *service) Create(ctx context.Context, source, idToken string) (Session, error) {
	id, err := s.resolveIdentity(ctx, source, idToken)
	if err != nil {
		return Session{}, err
	}

	token, err := newToken()
	if err != nil {
		return Session{}, apperr.Wrapf(apperr.KindInternal, err, "session: generate token")
	}

	sess := Session{Token: token, Identity: id, ExpiresAt: time.Now().Add(s.ttl)}
	if err := s.repo.Create(ctx, sess); err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s *service) resolveIdentity(ctx context.Context, source, idToken string) (identity.Identity, error) {
	switch source {
	case "line":
		if s.lineVerifier == nil {
			return identity.Identity{}, ErrLineAuthNotConfigured
		}
		return s.lineVerifier.Verify(ctx, idToken)
	case "demo":
		if !s.allowDemo {
			return identity.Identity{}, ErrDemoAuthDisabled
		}
		return identity.Identity{Source: "demo", ExternalID: "demo-user", DisplayName: "Demo User"}, nil
	default:
		return identity.Identity{}, identity.ErrInvalidToken
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
