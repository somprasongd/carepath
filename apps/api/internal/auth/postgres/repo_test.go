package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"carepath/apps/api/internal/auth"
	"carepath/apps/api/internal/platform/db"
)

// Integration test against a real Postgres. Requires the schema and seed
// data from infra/postgres/migrations; run `make migrate-up` first.
func newDB(t *testing.T) *db.DB {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test (run make up / migrate-up first)")
	}
	database, err := db.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(database.Close)
	return database
}

func TestSeedUsersAndRoles(t *testing.T) {
	repo := New(newDB(t))

	staff, err := repo.GetUserByUsername(context.Background(), "staff")
	if err != nil {
		t.Fatalf("GetUserByUsername staff: %v", err)
	}
	if staff.UserID != "user-staff" || !staff.IsActive || staff.FullName == "" {
		t.Fatalf("staff = %+v, want the active seeded demo user", staff)
	}
	if !auth.VerifyPassword("demo", staff.PasswordHash) {
		t.Fatal("seeded staff hash does not verify the documented demo password")
	}
	if len(staff.Roles) != 1 || staff.Roles[0] != auth.RoleStaff {
		t.Fatalf("staff.Roles = %v, want [STAFF]", staff.Roles)
	}

	admin, err := repo.GetUserByID(context.Background(), "user-admin")
	if err != nil {
		t.Fatalf("GetUserByID admin: %v", err)
	}
	if !admin.HasRole(auth.RoleAdmin) || !auth.VerifyPassword("demo", admin.PasswordHash) {
		t.Fatalf("admin = %+v, want an ADMIN whose demo password verifies", admin)
	}

	if _, err := repo.GetUserByUsername(context.Background(), "no-such-user"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("unknown user error = %v, want auth.ErrInvalidCredentials", err)
	}
}

func TestRefreshTokenLifecycle(t *testing.T) {
	repo := New(newDB(t))
	ctx := context.Background()

	// Unique per run: cleanup revokes rather than deletes, so a fixed raw
	// token would block every later local run on the same database with a
	// duplicate-key insert (CI always starts empty and never sees this).
	now := time.Now().UTC()
	stored := auth.RefreshToken{
		TokenHash: auth.HashToken(fmt.Sprintf("lifecycle-test-%d", now.UnixNano())),
		UserID:    "user-staff",
		IssuedAt:  now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	t.Cleanup(func() {
		_ = repo.RevokeRefreshToken(context.Background(), stored.TokenHash, time.Now())
	})
	if err := repo.CreateRefreshToken(ctx, stored); err != nil {
		t.Fatalf("CreateRefreshToken: %v", err)
	}

	got, err := repo.GetRefreshToken(ctx, stored.TokenHash)
	if err != nil {
		t.Fatalf("GetRefreshToken: %v", err)
	}
	if got.UserID != stored.UserID || !got.Usable(now) {
		t.Fatalf("stored token = %+v, want a usable token for %s", got, stored.UserID)
	}

	// Marking used succeeds exactly once — the second attempt must report
	// false so the service can treat it as reuse.
	if ok, err := repo.MarkRefreshTokenUsed(ctx, stored.TokenHash, now); err != nil || !ok {
		t.Fatalf("first MarkRefreshTokenUsed = %v, %v; want true, nil", ok, err)
	}
	if ok, err := repo.MarkRefreshTokenUsed(ctx, stored.TokenHash, now); err != nil || ok {
		t.Fatalf("second MarkRefreshTokenUsed = %v, %v; want false, nil", ok, err)
	}

	// Revoking the whole user's set makes every hash unusable.
	other := auth.RefreshToken{
		TokenHash: auth.HashToken(fmt.Sprintf("lifecycle-test-other-%d", now.UnixNano())),
		UserID:    "user-staff",
		IssuedAt:  now,
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if err := repo.CreateRefreshToken(ctx, other); err != nil {
		t.Fatalf("CreateRefreshToken other: %v", err)
	}
	if err := repo.RevokeUserRefreshTokens(ctx, "user-staff", now); err != nil {
		t.Fatalf("RevokeUserRefreshTokens: %v", err)
	}
	got, err = repo.GetRefreshToken(ctx, other.TokenHash)
	if err != nil {
		t.Fatalf("GetRefreshToken after revoke: %v", err)
	}
	if got.Usable(now) {
		t.Fatal("revoked token still usable")
	}
}
