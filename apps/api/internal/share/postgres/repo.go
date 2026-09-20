package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/share"
)

// Repo implements share.Repo on pgx. Every query goes through
// database.Querier(ctx) so it joins the ambient transaction when one is open.
type Repo struct {
	database *db.DB
}

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

func (r *Repo) Create(ctx context.Context, link share.ShareLink) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.visit_share_link (token_hash, visit_id, created_at, expires_at, revoked_at)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (token_hash) DO NOTHING`,
		link.TokenHash, link.VisitID, link.CreatedAt, link.ExpiresAt, link.RevokedAt)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "share: create link")
	}
	return nil
}

func (r *Repo) GetByTokenHash(ctx context.Context, tokenHash string) (share.ShareLink, error) {
	row := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT token_hash, visit_id, created_at, expires_at, revoked_at
		 FROM carepath.visit_share_link
		 WHERE token_hash = $1`, tokenHash)
	var link share.ShareLink
	if err := row.Scan(&link.TokenHash, &link.VisitID, &link.CreatedAt, &link.ExpiresAt, &link.RevokedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return share.ShareLink{}, apperr.New(apperr.KindNotFound, "share link not found")
		}
		return share.ShareLink{}, apperr.Wrapf(apperr.KindInternal, err, "share: get link")
	}
	return link, nil
}

func (r *Repo) CountActive(ctx context.Context, visitID string) (int, error) {
	var count int
	if err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT count(*) FROM carepath.visit_share_link
		 WHERE visit_id = $1 AND revoked_at IS NULL AND expires_at > now()`, visitID,
	).Scan(&count); err != nil {
		return 0, apperr.Wrapf(apperr.KindInternal, err, "share: count active links")
	}
	return count, nil
}

func (r *Repo) RevokeActive(ctx context.Context, visitID string) (int, error) {
	tag, err := r.database.Querier(ctx).Exec(ctx,
		`UPDATE carepath.visit_share_link
		 SET revoked_at = now()
		 WHERE visit_id = $1 AND revoked_at IS NULL AND expires_at > now()`, visitID)
	if err != nil {
		return 0, apperr.Wrapf(apperr.KindInternal, err, "share: revoke links")
	}
	return int(tag.RowsAffected()), nil
}
