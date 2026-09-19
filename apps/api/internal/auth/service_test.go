package auth

import (
	"context"
	"maps"
	"slices"
	"testing"
	"time"

	"carepath/apps/api/internal/platform/apperr"
)

// fakeRepo is an in-memory auth.Repo; txer runs fn directly (no real
// transaction semantics are under test here).
type fakeRepo struct {
	users   map[string]User         // by username
	byID    map[string]User         // by user id
	tokens  map[string]RefreshToken // by token_hash
	spent   map[string]bool         // hashes MarkRefreshTokenUsed rejected
	revoked []string                // user IDs whose sets were revoked
	now     time.Time
}

func newFakeRepo() *fakeRepo {
	hash, _ := HashPassword("demo")
	u := User{UserID: "user-1", Username: "somchai.r", PasswordHash: hash,
		FullName: "Somchai R", IsActive: true, Roles: []string{RoleStaff}}
	admin := User{UserID: "user-2", Username: "admin", PasswordHash: hash,
		FullName: "Admin", IsActive: true, Roles: []string{RoleAdmin}}
	inactive := User{UserID: "user-3", Username: "ghost", PasswordHash: hash,
		FullName: "Disabled", IsActive: false, Roles: []string{RoleStaff}}
	return &fakeRepo{
		users:  map[string]User{"somchai.r": u, "admin": admin, "ghost": inactive},
		byID:   map[string]User{"user-1": u, "user-2": admin, "user-3": inactive},
		tokens: map[string]RefreshToken{},
		now:    time.Now(),
	}
}

func (f *fakeRepo) GetUserByUsername(_ context.Context, username string) (User, error) {
	if u, ok := f.users[username]; ok {
		return u, nil
	}
	return User{}, ErrInvalidCredentials
}

func (f *fakeRepo) GetUserByID(_ context.Context, id string) (User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return User{}, ErrInvalidCredentials
}

func (f *fakeRepo) CreateRefreshToken(_ context.Context, t RefreshToken) error {
	f.tokens[t.TokenHash] = t
	return nil
}

func (f *fakeRepo) GetRefreshToken(_ context.Context, hash string) (RefreshToken, error) {
	if t, ok := f.tokens[hash]; ok {
		return t, nil
	}
	return RefreshToken{}, ErrInvalidCredentials
}

func (f *fakeRepo) MarkRefreshTokenUsed(_ context.Context, hash string, at time.Time) (bool, error) {
	t, ok := f.tokens[hash]
	if !ok || t.UsedAt != nil || t.RevokedAt != nil {
		return false, nil
	}
	t.UsedAt = &at
	f.tokens[hash] = t
	return true, nil
}

func (f *fakeRepo) RevokeRefreshToken(_ context.Context, hash string, at time.Time) error {
	if t, ok := f.tokens[hash]; ok && t.RevokedAt == nil {
		t.RevokedAt = &at
		f.tokens[hash] = t
	}
	return nil
}

func (f *fakeRepo) RevokeUserRefreshTokens(_ context.Context, userID string, at time.Time) error {
	f.revoked = append(f.revoked, userID)
	for h, t := range f.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &at
			f.tokens[h] = t
		}
	}
	return nil
}

// rollbackTxer behaves like the real Transactor: writes made inside fn are
// kept on success and discarded when fn fails. Without this the reuse
// detection test below passes even when the revocation is rolled back with
// the rejection — exactly the bug live verification caught.
type rollbackTxer struct{ repo *fakeRepo }

func (t rollbackTxer) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	users := maps.Clone(t.repo.users)
	byID := maps.Clone(t.repo.byID)
	tokens := maps.Clone(t.repo.tokens)
	revoked := slices.Clone(t.repo.revoked)
	if err := fn(ctx); err != nil {
		t.repo.users, t.repo.byID, t.repo.tokens, t.repo.revoked = users, byID, tokens, revoked
		return err
	}
	return nil
}

func newTestService(repo *fakeRepo) Service {
	return NewService(repo, rollbackTxer{repo},
		NewTokenIssuer([]byte("a-test-secret-32-bytes-long!!"), 15*time.Minute), 7*24*time.Hour)
}

func TestLoginSuccess(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	pair, err := svc.Login(context.Background(), "somchai.r", "demo")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatal("empty token in pair")
	}
	if !pair.Identity.HasRole(RoleStaff) || pair.Identity.HasRole(RoleAdmin) {
		t.Fatalf("identity roles = %v", pair.Identity.Roles)
	}
	if p, err := svc.ParseAccessToken(pair.AccessToken); err != nil || p.Username != "somchai.r" {
		t.Fatalf("issued access token does not parse: %v %+v", err, p)
	}
	// The persisted refresh token is the hash, never the raw token.
	if _, ok := repo.tokens[pair.RefreshToken]; ok {
		t.Fatal("raw refresh token was persisted")
	}
	if _, ok := repo.tokens[HashToken(pair.RefreshToken)]; !ok {
		t.Fatal("refresh token hash missing from the store")
	}
}

func TestLoginFailuresAreUniform(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	cases := []struct{ name, user, pass string }{
		{"wrong password", "somchai.r", "nope"},
		{"unknown user", "nobody", "demo"},
		{"unknown user wrong password", "nobody", "nope"},
		{"deactivated", "ghost", "demo"},
	}
	for _, tc := range cases {
		_, err := svc.Login(context.Background(), tc.user, tc.pass)
		if err == nil {
			t.Fatalf("%s: login succeeded", tc.name)
		}
		if apperr.KindOf(err) != apperr.KindUnauthorized || err.Error() != ErrInvalidCredentials.Error() {
			t.Fatalf("%s: error = %v, want the uniform invalid-credentials 401", tc.name, err)
		}
	}
}

func TestRefreshRotatesAndSpends(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	first, err := svc.Login(context.Background(), "somchai.r", "demo")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	second, err := svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if second.AccessToken == first.AccessToken || second.RefreshToken == first.RefreshToken {
		t.Fatal("rotation must issue fresh tokens")
	}
	if _, err := svc.ParseAccessToken(second.AccessToken); err != nil {
		t.Fatalf("rotated access token invalid: %v", err)
	}
}

func TestRefreshReuseRevokesWholeSet(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	first, _ := svc.Login(context.Background(), "somchai.r", "demo")
	second, err := svc.Refresh(context.Background(), first.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	// Replaying the spent token must fail…
	if _, err := svc.Refresh(context.Background(), first.RefreshToken); err == nil {
		t.Fatal("spent refresh token was accepted")
	}
	// …and burn the still-fresh rotated token with it.
	if _, err := svc.Refresh(context.Background(), second.RefreshToken); err == nil {
		t.Fatal("token from the abused set still works after reuse detection")
	}
	if len(repo.revoked) == 0 || repo.revoked[len(repo.revoked)-1] != "user-1" {
		t.Fatalf("revocations recorded = %v", repo.revoked)
	}
}

func TestRefreshExpiredTokenFails(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	first, _ := svc.Login(context.Background(), "somchai.r", "demo")
	stored := repo.tokens[HashToken(first.RefreshToken)]
	stored.ExpiresAt = repo.now.Add(-time.Hour)
	repo.tokens[HashToken(first.RefreshToken)] = stored
	if _, err := svc.Refresh(context.Background(), first.RefreshToken); err == nil {
		t.Fatal("expired refresh token was accepted")
	}
}

func TestLogoutMakesTokenUnusable(t *testing.T) {
	repo := newFakeRepo()
	svc := newTestService(repo)
	pair, _ := svc.Login(context.Background(), "somchai.r", "demo")
	if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(context.Background(), pair.RefreshToken); err == nil {
		t.Fatal("logged-out refresh token was accepted")
	}
	// Idempotent: an unknown token is still a successful logout.
	if err := svc.Logout(context.Background(), "not-a-token"); err != nil {
		t.Fatalf("logout of unknown token: %v", err)
	}
}
