package visitlink_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"carepath/apps/api/internal/his"
	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/visitlink"
)

// The link's policy (#136) pinned with fakes: mint is idempotent, rotation
// kills the previous token, the lifetime policy is ACTIVE/COMPLETED-grace/
// CANCELLED, the token is never stored raw, and a rotated VISIT_LINK_SECRET
// revokes every printed link.

type fakeRepo struct {
	visits  map[string]string // visitID → status
	rows    map[string]string // visitID → token hash
	version map[string]int    // visitID → version
	done    map[string]*time.Time
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		visits: map[string]string{}, rows: map[string]string{},
		version: map[string]int{}, done: map[string]*time.Time{},
	}
}

func (f *fakeRepo) Current(_ context.Context, visitID string) (string, int, bool, error) {
	if _, ok := f.visits[visitID]; !ok {
		return "", 0, false, nil
	}
	return f.rows[visitID], f.version[visitID], true, nil
}

func (f *fakeRepo) Store(_ context.Context, visitID, hash string, version int) error {
	f.rows[visitID] = hash
	f.version[visitID] = version
	return nil
}

func (f *fakeRepo) Resolve(_ context.Context, hash string) (string, string, int, *time.Time, error) {
	for visitID, h := range f.rows {
		if h == hash {
			return visitID, f.visits[visitID], f.version[visitID], f.done[visitID], nil
		}
	}
	return "", "", 0, nil, visitlink.ErrNotFound
}

type fakeClaims struct {
	claimed map[[2]string]string // (source, externalID) → visitID
}

func (f *fakeClaims) Claim(_ context.Context, id identity.Identity, visitID string) error {
	if f.claimed == nil {
		f.claimed = map[[2]string]string{}
	}
	f.claimed[[2]string{id.Source, id.ExternalID}] = visitID
	return nil
}

func newService(repo *fakeRepo, secret string, grace time.Duration) visitlink.Service {
	return visitlink.NewService(repo, &fakeClaims{}, []byte(secret), "https://x.test", grace,
		slog.Default())
}

func tokenOf(t *testing.T, url string) string {
	t.Helper()
	raw, ok := strings.CutPrefix(url, "https://x.test/patient/journey#vt=")
	if !ok {
		t.Fatalf("url %q is not the expected fragment form", url)
	}
	return raw
}

func TestMintIsIdempotentUntilRotate(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-1"] = his.VisitActive
	svc := newService(repo, "secret-1", 30*time.Minute)

	first, err := svc.Mint(context.Background(), "VISIT-1", false)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	// Only the hash is persisted — the raw token exists in the response alone.
	for _, h := range repo.rows {
		if h == tokenOf(t, first.VisitURL) {
			t.Fatal("raw token stored in the repo")
		}
	}
	again, err := svc.Mint(context.Background(), "VISIT-1", false)
	if err != nil {
		t.Fatalf("re-mint: %v", err)
	}
	if again.VisitURL != first.VisitURL {
		t.Fatalf("re-mint returned %q, want the same URL %q", again.VisitURL, first.VisitURL)
	}

	rotated, err := svc.Mint(context.Background(), "VISIT-1", true)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if rotated.VisitURL == first.VisitURL {
		t.Fatal("rotate returned the same URL")
	}
}

func TestMintUnknownVisitIsJourneyNotFound(t *testing.T) {
	svc := newService(newFakeRepo(), "secret-1", 30*time.Minute)
	_, err := svc.Mint(context.Background(), "VISIT-NOPE", false)
	if err != visitlink.ErrNotFound || !strings.Contains(err.Error(), "journey not found") {
		t.Fatalf("mint unknown: %v, want the journey 404", err)
	}
}

func redeemOK(t *testing.T, svc visitlink.Service, token string) string {
	t.Helper()
	visitID, err := svc.Redeem(context.Background(),
		identity.Identity{Source: "line", ExternalID: "U-1"}, token)
	if err != nil {
		t.Fatalf("redeem: %v", err)
	}
	return visitID
}

func TestRedeemLifecycle(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-1"] = his.VisitActive
	svc := newService(repo, "secret-1", 30*time.Minute)

	link, err := svc.Mint(context.Background(), "VISIT-1", false)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	raw := tokenOf(t, link.VisitURL)

	if got := redeemOK(t, svc, raw); got != "VISIT-1" {
		t.Fatalf("redeem returned %q, want VISIT-1", got)
	}
	// Redeem is repeatable: re-opening the link from LINE chat history
	// works, and the claim mint is idempotent for the same identity.
	redeemOK(t, svc, raw)

	// Rotation kills the previous token even for someone who saved it.
	rotated, err := svc.Mint(context.Background(), "VISIT-1", true)
	if err != nil {
		t.Fatalf("rotate: %v", err)
	}
	if _, err := svc.Redeem(context.Background(), identity.Identity{Source: "line", ExternalID: "U-1"}, raw); err != visitlink.ErrNotFound {
		t.Fatalf("redeem old token after rotate: %v, want 404", err)
	}
	redeemOK(t, svc, tokenOf(t, rotated.VisitURL))
}

func TestRedeemMalformedAndUnknownAreTheSame404(t *testing.T) {
	svc := newService(newFakeRepo(), "secret-1", 30*time.Minute)
	id := identity.Identity{Source: "line", ExternalID: "U-1"}
	for _, raw := range []string{"", "short", "not-base64-$$$-at-all--------", strings.Repeat("A", 22)} {
		if _, err := svc.Redeem(context.Background(), id, raw); err != visitlink.ErrNotFound {
			t.Fatalf("redeem %q: %v, want the journey 404", raw, err)
		}
	}
}

func TestRedeemLifetimePolicy(t *testing.T) {
	completedTenAgo := time.Now().Add(-10 * time.Minute)
	completedHourAgo := time.Now().Add(-61 * time.Minute)
	cases := []struct {
		name        string
		status      string
		completedAt *time.Time
		grace       time.Duration
		want        bool
	}{
		{"active", his.VisitActive, nil, 30 * time.Minute, true},
		{"cancelled", his.VisitCancelled, nil, 30 * time.Minute, false},
		{"completed in grace", his.VisitCompleted, &completedTenAgo, 30 * time.Minute, true},
		{"completed past grace", his.VisitCompleted, &completedHourAgo, 30 * time.Minute, false},
		{"completed pre-000022 row", his.VisitCompleted, nil, 30 * time.Minute, false},
		{"completed with zero grace", his.VisitCompleted, &completedTenAgo, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := newFakeRepo()
			repo.visits["VISIT-1"] = tc.status
			repo.done["VISIT-1"] = tc.completedAt
			svc := newService(repo, "secret-1", tc.grace)
			link, err := svc.Mint(context.Background(), "VISIT-1", false)
			if err != nil {
				t.Fatalf("mint: %v", err)
			}
			_, err = svc.Redeem(context.Background(),
				identity.Identity{Source: "line", ExternalID: "U-1"}, tokenOf(t, link.VisitURL))
			if tc.want && err != nil {
				t.Fatalf("redeem: %v, want success", err)
			}
			if !tc.want && err != visitlink.ErrNotFound {
				t.Fatalf("redeem: %v, want the journey 404", err)
			}
		})
	}
}

func TestSecretRotationRevokesPrintedLinks(t *testing.T) {
	repo := newFakeRepo()
	repo.visits["VISIT-1"] = his.VisitActive
	before := newService(repo, "old-secret", 30*time.Minute)
	link, err := before.Mint(context.Background(), "VISIT-1", false)
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	raw := tokenOf(t, link.VisitURL)

	// Same repo, new secret: the hash still resolves, but the HMAC proof
	// fails — an operator rotating VISIT_LINK_SECRET kills every printed
	// link at once, by design.
	after := newService(repo, "new-secret", 30*time.Minute)
	if _, err := after.Redeem(context.Background(), identity.Identity{Source: "line", ExternalID: "U-1"}, raw); err != visitlink.ErrNotFound {
		t.Fatalf("redeem under new secret: %v, want 404", err)
	}
	// And the next mint under the new secret issues a fresh version.
	fresh, err := after.Mint(context.Background(), "VISIT-1", false)
	if err != nil {
		t.Fatalf("re-mint under new secret: %v", err)
	}
	if fresh.VisitURL == link.VisitURL {
		t.Fatal("mint under a new secret returned the old-secret URL")
	}
}
