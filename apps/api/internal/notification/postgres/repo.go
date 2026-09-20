// Package postgres is the notification module's persistence adapter over
// pgx: the exactly-once claim, the opt-out preference, and the visit → LINE
// user resolution from the claim table. Queries resolve their connection
// from the ambient context like every other module's repo.
package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"carepath/apps/api/internal/notification"
	"carepath/apps/api/internal/platform/apperr"
	"carepath/apps/api/internal/platform/db"
)

type Repo struct {
	database *db.DB
}

var _ notification.Store = (*Repo)(nil)

func New(database *db.DB) *Repo {
	return &Repo{database: database}
}

// SendOnce claims the (visit, step) slot. ON CONFLICT DO NOTHING plus
// RETURNING makes the claim atomic: only the insert that landed returns a
// row, so concurrent sweeps cannot both believe they may send.
func (r *Repo) SendOnce(ctx context.Context, visitID, stepKey, channel string) (bool, error) {
	var one int
	err := r.database.Querier(ctx).QueryRow(ctx, `
		INSERT INTO carepath.queue_notification (visit_id, step_key, channel)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING 1`,
		visitID, stepKey, channel).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, apperr.Wrapf(apperr.KindInternal, err, "notification: claim send for visit %q step %q", visitID, stepKey)
	}
	return true, nil
}

// Enabled reports the opt-out state; no row means enabled (ADR-0013 §4), so
// existing visits need no backfill.
func (r *Repo) Enabled(ctx context.Context, visitID string) (bool, error) {
	enabled := true
	err := r.database.Querier(ctx).QueryRow(ctx, `
		SELECT enabled FROM carepath.visit_notification_pref WHERE visit_id = $1`,
		visitID).Scan(&enabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, apperr.Wrapf(apperr.KindInternal, err, "notification: read pref for visit %q", visitID)
	}
	return enabled, nil
}

// SetEnabled upserts the preference. The visit_id FK rejects a visit that
// was never projected, mapped to the journey-shaped 404.
func (r *Repo) SetEnabled(ctx context.Context, visitID string, enabled bool) error {
	_, err := r.database.Querier(ctx).Exec(ctx, `
		INSERT INTO carepath.visit_notification_pref (visit_id, enabled)
		VALUES ($1, $2)
		ON CONFLICT (visit_id) DO UPDATE
		SET enabled = EXCLUDED.enabled, updated_at = now()`,
		visitID, enabled)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return notification.ErrVisitNotFound
		}
		return apperr.Wrapf(apperr.KindInternal, err, "notification: write pref for visit %q", visitID)
	}
	return nil
}

// Recipients resolves the visit's LINE user id from the claim table (#96):
// the most recent line-source identity that claimed the visit. Demo claims
// (source=demo) are never LINE recipients — ok=false, not an error.
type Recipients struct {
	database *db.DB
}

var _ notification.Recipients = (*Recipients)(nil)

func NewRecipients(database *db.DB) *Recipients {
	return &Recipients{database: database}
}

func (r *Recipients) LineUser(ctx context.Context, visitID string) (string, bool, error) {
	var userID string
	err := r.database.Querier(ctx).QueryRow(ctx, `
		SELECT external_id FROM carepath.patient_visit_claim
		WHERE visit_id = $1 AND source = 'line'
		ORDER BY claimed_at DESC
		LIMIT 1`,
		visitID).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, apperr.Wrapf(apperr.KindInternal, err, "notification: resolve recipient for visit %q", visitID)
	}
	return userID, true, nil
}
