package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

// TokenPair is the credential set issued by login and refresh: the client
// gets both raw tokens once; the server persists only the refresh hash.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
	Identity         Principal
}

// Service is the only entry point other modules may call.
type Service interface {
	// Login verifies the username/password pair and mints a fresh pair.
	// Every failure — unknown user, wrong password, deactivated account —
	// returns the same ErrInvalidCredentials, and an unknown username still
	// pays for one argon2 verify.
	Login(ctx context.Context, username, password string) (TokenPair, error)
	// Refresh spends the presented refresh token and issues a new pair.
	// Presenting an already-spent token revokes every refresh token of that
	// user (reuse = leak, ADR-0010 §4) and fails.
	Refresh(ctx context.Context, refreshToken string) (TokenPair, error)
	// Logout revokes the presented refresh token; unknown tokens are a
	// no-op so a client can always clear its own state.
	Logout(ctx context.Context, refreshToken string) error
	// ParseAccessToken resolves an access token to its principal without a
	// database round trip.
	ParseAccessToken(token string) (Principal, error)
	// AccessTokenTTL echoes the configured lifetime for API responses.
	AccessTokenTTL() time.Duration
}

type service struct {
	repo       Repo
	tx         db.Transactor
	tokens     *TokenIssuer
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(repo Repo, tx db.Transactor, tokens *TokenIssuer, refreshTTL time.Duration) Service {
	return &service{repo: repo, tx: tx, tokens: tokens, refreshTTL: refreshTTL, now: time.Now}
}

func (s *service) Login(ctx context.Context, username, password string) (TokenPair, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if apperr.KindOf(err) != apperr.KindUnauthorized {
			return TokenPair{}, err // infrastructure failure, not a login failure
		}
		// Equalize timing with the real verify, then fail identically.
		VerifyPassword(password, dummyHash)
		return TokenPair{}, ErrInvalidCredentials
	}
	if !user.IsActive || !VerifyPassword(password, user.PasswordHash) {
		return TokenPair{}, ErrInvalidCredentials
	}
	return s.issuePair(ctx, principalOf(user))
}

func (s *service) Refresh(ctx context.Context, refreshToken string) (TokenPair, error) {
	var pair TokenPair
	// rejected carries a business refusal whose side effects must still
	// commit: reuse detection revokes the user's whole token set, and
	// returning the refusal from the transaction fn would roll the
	// revocation back with it.
	var rejected error
	err := s.tx.WithinTransaction(ctx, func(ctx context.Context) error {
		now := s.now()
		stored, err := s.repo.GetRefreshToken(ctx, HashToken(refreshToken))
		if err != nil {
			rejected = ErrInvalidCredentials
			return nil
		}
		if stored.RevokedAt != nil {
			rejected = ErrInvalidCredentials
			return nil
		}
		if stored.UsedAt != nil {
			// A spent token replaying means it leaked: burn the whole set.
			if err := s.repo.RevokeUserRefreshTokens(ctx, stored.UserID, now); err != nil {
				return err
			}
			rejected = ErrInvalidCredentials
			return nil
		}
		if !stored.ExpiresAt.After(now) {
			rejected = ErrInvalidCredentials
			return nil
		}
		spent, err := s.repo.MarkRefreshTokenUsed(ctx, stored.TokenHash, now)
		if err != nil {
			return err
		}
		if !spent {
			// A concurrent rotation won the race — same story as reuse.
			if err := s.repo.RevokeUserRefreshTokens(ctx, stored.UserID, now); err != nil {
				return err
			}
			rejected = ErrInvalidCredentials
			return nil
		}
		user, err := s.repo.GetUserByID(ctx, stored.UserID)
		if err != nil || !user.IsActive {
			rejected = ErrInvalidCredentials
			return nil
		}
		var issueErr error
		pair, issueErr = s.issuePair(ctx, principalOf(user))
		return issueErr
	})
	if err != nil {
		return TokenPair{}, err
	}
	if rejected != nil {
		return TokenPair{}, rejected
	}
	return pair, nil
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	return s.repo.RevokeRefreshToken(ctx, HashToken(refreshToken), s.now())
}

func (s *service) ParseAccessToken(token string) (Principal, error) {
	return s.tokens.ParseAccessToken(token)
}

func (s *service) AccessTokenTTL() time.Duration { return s.tokens.TTL() }

// issuePair mints the access token and the rotating refresh token; the
// refresh hash is persisted inside the ambient transaction when one is open
// (refresh), or standalone (login).
func (s *service) issuePair(ctx context.Context, p Principal) (TokenPair, error) {
	access, expiresAt, err := s.tokens.IssueAccessToken(p)
	if err != nil {
		return TokenPair{}, apperr.Wrapf(apperr.KindInternal, err, "auth: issue access token")
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return TokenPair{}, apperr.Wrapf(apperr.KindInternal, err, "auth: generate refresh token")
	}
	refreshToken := hex.EncodeToString(raw)
	now := s.now()
	if err := s.repo.CreateRefreshToken(ctx, RefreshToken{
		TokenHash: HashToken(refreshToken),
		UserID:    p.UserID,
		IssuedAt:  now,
		ExpiresAt: now.Add(s.refreshTTL),
	}); err != nil {
		return TokenPair{}, err
	}
	return TokenPair{
		AccessToken:      access,
		RefreshToken:     refreshToken,
		ExpiresAt:        expiresAt,
		RefreshExpiresAt: now.Add(s.refreshTTL),
		Identity:         p,
	}, nil
}

func principalOf(u User) Principal {
	return Principal{UserID: u.UserID, Username: u.Username, DisplayName: u.FullName, Roles: u.Roles}
}

// HashToken is the at-rest form of a refresh token: the raw 256-bit token
// needs no slow hash, only to not be stored verbatim (ADR-0010 §4).
func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
