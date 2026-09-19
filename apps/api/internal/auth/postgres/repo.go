// Package postgres is the auth module's persistence adapter on pgx.
package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/auth"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type Repo struct {
	database *db.DB
}

var _ auth.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

func (r *Repo) GetUserByUsername(ctx context.Context, username string) (auth.User, error) {
	return r.scanUser(ctx,
		`SELECT u.user_id, u.username, u.password_hash, u.full_name, u.is_active,
		        coalesce(array_agg(r.code ORDER BY r.code) FILTER (WHERE r.code IS NOT NULL), '{}')
		 FROM carepath.app_user u
		 LEFT JOIN carepath.user_role ur ON ur.user_id = u.user_id
		 LEFT JOIN carepath.role r ON r.role_id = ur.role_id
		 WHERE u.username = $1
		 GROUP BY u.user_id`, username)
}

func (r *Repo) GetUserByID(ctx context.Context, userID string) (auth.User, error) {
	return r.scanUser(ctx,
		`SELECT u.user_id, u.username, u.password_hash, u.full_name, u.is_active,
		        coalesce(array_agg(r.code ORDER BY r.code) FILTER (WHERE r.code IS NOT NULL), '{}')
		 FROM carepath.app_user u
		 LEFT JOIN carepath.user_role ur ON ur.user_id = u.user_id
		 LEFT JOIN carepath.role r ON r.role_id = ur.role_id
		 WHERE u.user_id = $1
		 GROUP BY u.user_id`, userID)
}

func (r *Repo) scanUser(ctx context.Context, sql string, arg any) (auth.User, error) {
	var u auth.User
	err := r.database.Querier(ctx).QueryRow(ctx, sql, arg).
		Scan(&u.UserID, &u.Username, &u.PasswordHash, &u.FullName, &u.IsActive, &u.Roles)
	if errors.Is(err, pgx.ErrNoRows) {
		// Not-found is a login failure, and login failures are uniform.
		return auth.User{}, auth.ErrInvalidCredentials
	}
	if err != nil {
		return auth.User{}, apperr.Wrapf(apperr.KindInternal, err, "auth: load user")
	}
	return u, nil
}

func (r *Repo) CreateRefreshToken(ctx context.Context, token auth.RefreshToken) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.refresh_token (token_hash, user_id, issued_at, expires_at)
		 VALUES ($1, $2, $3, $4)`,
		token.TokenHash, token.UserID, token.IssuedAt, token.ExpiresAt,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "auth: store refresh token")
	}
	return nil
}

func (r *Repo) GetRefreshToken(ctx context.Context, tokenHash string) (auth.RefreshToken, error) {
	var t auth.RefreshToken
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT token_hash, user_id, issued_at, expires_at, used_at, revoked_at
		 FROM carepath.refresh_token WHERE token_hash = $1`, tokenHash,
	).Scan(&t.TokenHash, &t.UserID, &t.IssuedAt, &t.ExpiresAt, &t.UsedAt, &t.RevokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.RefreshToken{}, auth.ErrInvalidCredentials
	}
	if err != nil {
		return auth.RefreshToken{}, apperr.Wrapf(apperr.KindInternal, err, "auth: load refresh token")
	}
	return t, nil
}

func (r *Repo) MarkRefreshTokenUsed(ctx context.Context, tokenHash string, at time.Time) (bool, error) {
	tag, err := r.database.Querier(ctx).Exec(ctx,
		`UPDATE carepath.refresh_token SET used_at = $2
		 WHERE token_hash = $1 AND used_at IS NULL AND revoked_at IS NULL`,
		tokenHash, at,
	)
	if err != nil {
		return false, apperr.Wrapf(apperr.KindInternal, err, "auth: spend refresh token")
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) RevokeRefreshToken(ctx context.Context, tokenHash string, at time.Time) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`UPDATE carepath.refresh_token SET revoked_at = $2
		 WHERE token_hash = $1 AND revoked_at IS NULL`,
		tokenHash, at,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "auth: revoke refresh token")
	}
	return nil
}

func (r *Repo) RevokeUserRefreshTokens(ctx context.Context, userID string, at time.Time) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`UPDATE carepath.refresh_token SET revoked_at = $2
		 WHERE user_id = $1 AND revoked_at IS NULL`,
		userID, at,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "auth: revoke user refresh tokens")
	}
	return nil
}
