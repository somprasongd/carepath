// Claim persistence for #96: which identity may read which visit.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"carepath/apps/api/internal/identity"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/session"
)

type ClaimRepo struct {
	database *db.DB
}

var _ session.ClaimRepo = (*ClaimRepo)(nil)

func NewClaimRepo(database *db.DB) *ClaimRepo {
	return &ClaimRepo{database: database}
}

// Claim inserts the ownership row. ON CONFLICT DO NOTHING makes a repeat
// claim by the same identity a no-op; the visit_id FK rejects a visit that
// was never projected, mapped to the journey-shaped 404.
func (r *ClaimRepo) Claim(ctx context.Context, id identity.Identity, visitID string) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.patient_visit_claim (source, external_id, visit_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT DO NOTHING`,
		id.Source, id.ExternalID, visitID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return session.ErrVisitNotFound
		}
		return apperr.Wrapf(apperr.KindInternal, err, "session: claim visit")
	}
	return nil
}

func (r *ClaimRepo) HasClaimed(ctx context.Context, id identity.Identity, visitID string) (bool, error) {
	var one int
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT 1 FROM carepath.patient_visit_claim
		 WHERE source = $1 AND external_id = $2 AND visit_id = $3`,
		id.Source, id.ExternalID, visitID,
	).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, apperr.Wrapf(apperr.KindInternal, err, "session: check claim")
	}
	return true, nil
}
