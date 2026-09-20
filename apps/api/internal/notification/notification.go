// Package notification implements FR-21 (#104, ADR-0013): notify the patient
// when their queue position is approaching. It is the project's first
// outbound integration, so the channel is a port — a no-op adapter runs by
// default, and the LINE Messaging API adapter activates only when the
// operator supplies a channel token.
package notification

import (
	"context"

	"carepath/apps/api/internal/platform/apperr"
)

// ErrVisitNotFound answers a preference read for a visit that was never
// projected — the same journey-shaped 404 the other patient surfaces give,
// so existence never leaks.
var ErrVisitNotFound = apperr.New(apperr.KindNotFound, "journey not found")

// Notifier is the outbound channel port (ADR-0013 §1): deliver one text
// message to one recipient. Recipient is opaque to the engine — the LINE
// adapter treats it as a user id; a future adapter as a phone number or
// address. Implementations must be safe for concurrent use.
type Notifier interface {
	// Channel names the adapter for the delivery record (queue_notification
	// .channel), e.g. "noop" or "line".
	Channel() string
	Notify(ctx context.Context, recipient, text string) error
}

// Recipients resolves the visit's LINE user id from the visit-claim table
// (#96): the identity that claimed the visit. Demo-mode visits (no LINE
// identity) resolve no recipient — ok=false, not an error.
type Recipients interface {
	LineUser(ctx context.Context, visitID string) (userID string, ok bool, err error)
}

// Store is the notification module's persistence port: the exactly-once
// claim (#104 AC) and the patient's opt-out preference.
type Store interface {
	// SendOnce claims the (visit, step) notification slot. It returns true
	// when this call won the claim — the only caller allowed to send.
	SendOnce(ctx context.Context, visitID, stepKey, channel string) (bool, error)
	// Enabled reports the visit's opt-out state; no row means enabled.
	Enabled(ctx context.Context, visitID string) (bool, error)
	// SetEnabled upserts the visit's opt-out state.
	SetEnabled(ctx context.Context, visitID string, enabled bool) error
}

// QueueSource is the slice of the journey module the sweep needs: the queue
// picture (#101) and the active visits with their recommended step (#103).
// Narrow on purpose — the sweep must not reach into plans or transitions.
type QueueSource interface {
	// ActiveVisits returns ACTIVE journeys that currently have a
	// recommended step bound to a service point.
	ActiveVisits(ctx context.Context) ([]ActiveVisit, error)
	// Queue returns the waiting-ahead count per actionable step for one
	// visit (FR-17 shape, minus everything the criterion does not need).
	Queue(ctx context.Context, visitID string) ([]QueueStep, error)
}

// ActiveVisit is one sweep candidate: the visit plus the recommended step
// the notification would be about — the same single primary action the
// patient screen shows, so a message never contradicts the app.
type ActiveVisit struct {
	VisitID      string
	StepKey      string
	ServicePoint string
}

// QueueStep is the sweep's view of one actionable step's queue position.
type QueueStep struct {
	StepKey      string
	WaitingAhead int
}

// Pref is the wire shape of GET/PUT /api/v1/journeys/{visitId}/notifications.
type Pref struct {
	VisitID string `json:"visitId"`
	Enabled bool   `json:"enabled"`
}
