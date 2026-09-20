// Persistence for #136: the visit's link token row and the projection
// state the lifetime policy reads.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/visitlink"
)

type Repo struct {
	database *db.DB
}

var _ visitlink.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

func (r *Repo) Current(ctx context.Context, visitID string) (string, int, bool, error) {
	// The token table is joined to journey_visit not for its columns but
	// for existence: a visit CarePath never projected has no link either.
	// A projected visit without a minted token reads as hash ""/version 0.
	var hash string
	var version int
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT COALESCE(t.token_hash, ''), COALESCE(t.version, 0)
		 FROM carepath.journey_visit v
		 LEFT JOIN carepath.visit_access_token t ON t.visit_id = v.visit_id
		 WHERE v.visit_id = $1`, visitID,
	).Scan(&hash, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, apperr.Wrapf(apperr.KindInternal, err, "visitlink: read token of %s", visitID)
	}
	return hash, version, true, nil
}

func (r *Repo) Store(ctx context.Context, visitID, tokenHash string, version int) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.visit_access_token (visit_id, version, token_hash)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (visit_id) DO UPDATE
		 SET version = EXCLUDED.version,
		     token_hash = EXCLUDED.token_hash,
		     generated_at = now()`,
		visitID, version, tokenHash,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "visitlink: store token of %s", visitID)
	}
	return nil
}

func (r *Repo) Resolve(ctx context.Context, tokenHash string) (string, string, int, *time.Time, error) {
	var visitID, status string
	var version int
	var completedAt *time.Time
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT v.visit_id, v.status, t.version, v.completed_at
		 FROM carepath.visit_access_token t
		 JOIN carepath.journey_visit v ON v.visit_id = t.visit_id
		 WHERE t.token_hash = $1`, tokenHash,
	).Scan(&visitID, &status, &version, &completedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", 0, nil, visitlink.ErrNotFound
	}
	if err != nil {
		return "", "", 0, nil, apperr.Wrapf(apperr.KindInternal, err, "visitlink: resolve token")
	}
	return visitID, status, version, completedAt, nil
}
