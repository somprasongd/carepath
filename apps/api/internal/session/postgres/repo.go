// Package postgres is the session module's persistence adapter over pgx.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
	"carepath/apps/api/internal/session"
)

type Repo struct {
	database *db.DB
}

var _ session.Repo = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

func (r *Repo) Create(ctx context.Context, s session.Session) error {
	_, err := r.database.Querier(ctx).Exec(ctx,
		`INSERT INTO carepath.patient_session (token, source, external_id, display_name, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		s.Token, s.Identity.Source, s.Identity.ExternalID, nullable(s.Identity.DisplayName), s.ExpiresAt,
	)
	if err != nil {
		return apperr.Wrapf(apperr.KindInternal, err, "session: create")
	}
	return nil
}

func (r *Repo) Get(ctx context.Context, token string) (session.Session, error) {
	var (
		s           session.Session
		displayName *string
	)
	err := r.database.Querier(ctx).QueryRow(ctx,
		`SELECT token, source, external_id, display_name, expires_at
		 FROM carepath.patient_session
		 WHERE token = $1 AND expires_at > now()`,
		token,
	).Scan(&s.Token, &s.Identity.Source, &s.Identity.ExternalID, &displayName, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return session.Session{}, session.ErrNotFound
	}
	if err != nil {
		return session.Session{}, apperr.Wrapf(apperr.KindInternal, err, "session: get")
	}
	if displayName != nil {
		s.Identity.DisplayName = *displayName
	}
	return s, nil
}

func nullable(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
