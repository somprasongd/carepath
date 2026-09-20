package postgres

import (
	"context"
	"testing"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
)

// The claim repo's one non-obvious behaviour: the visit_id FK doubles as
// existence validation, mapped to the journey-shaped 404 (#96).
func TestClaimRepo(t *testing.T) {
	database := newDB(t)
	repo := NewClaimRepo(database)
	ctx := context.Background()

	const visitID = "VISIT-TEST-CLAIM-1"
	id := identity.Identity{Source: "line", ExternalID: "U-CLAIM-TEST"}

	cleanup := func() {
		_, _ = database.Querier(context.Background()).Exec(context.Background(),
			`DELETE FROM carepath.journey_visit WHERE visit_id = $1`, visitID)
	}
	cleanup()
	t.Cleanup(cleanup)

	// Claiming a visit that was never projected is the journey 404.
	err := repo.Claim(ctx, id, visitID)
	if apperr.KindOf(err) != apperr.KindNotFound {
		t.Fatalf("Claim unknown visit: err = %v, want KindNotFound", err)
	}

	if _, err := database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.journey_visit (visit_id, patient_ref, status)
		 VALUES ($1, 'PATIENT-CLAIM-TEST', 'ACTIVE')`, visitID,
	); err != nil {
		t.Fatalf("seed visit: %v", err)
	}

	if err := repo.Claim(ctx, id, visitID); err != nil {
		t.Fatalf("Claim: %v", err)
	}
	// Idempotent for the same identity.
	if err := repo.Claim(ctx, id, visitID); err != nil {
		t.Fatalf("re-Claim: %v", err)
	}

	ok, err := repo.HasClaimed(ctx, id, visitID)
	if err != nil || !ok {
		t.Fatalf("HasClaimed after claim: %v %v, want true nil", ok, err)
	}
	other := identity.Identity{Source: "line", ExternalID: "U-OTHER"}
	ok, err = repo.HasClaimed(ctx, other, visitID)
	if err != nil || ok {
		t.Fatalf("HasClaimed other identity: %v %v, want false nil", ok, err)
	}
}
