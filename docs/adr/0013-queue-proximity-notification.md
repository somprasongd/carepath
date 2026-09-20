# ADR-0013: Queue-Proximity Notification via an Outbound Channel Port

- Status: Accepted
- Date: 2026-09-20
- Relates to: FR-21 (#104), [ADR-0004](0004-location-provider-abstraction.md) (provider port pattern), [ADR-0009](0009-carepath-owns-journey-plan.md) (CarePath recomputes plans), [ADR-0010](0010-staff-auth-jwt-argon2.md) (identity separation), [ADR-0012](0012-client-owned-display-text.md) (who owns patient-facing words)

## Context

FR-21 (Should · S6): notify a patient when their queue position is
approaching, so they can wait elsewhere without fear of losing their turn.
This is the project's **first outbound integration** — every message CarePath
has produced so far stayed inside its own HTTP responses.

The constraints that shape the decision:

- The patient app lives inside LINE (LIFF), so **LINE Messaging API push**
  is the only production channel worth building for the MVP. Web Push does
  not work inside LINE's in-app browser.
- A Messaging API channel is a **separate credential set** from the LIFF /
  LINE Login channel CarePath already reads (`LINE_CHANNEL_ID`): it needs its
  own channel and a long-lived channel access token, and **push messages are
  billed**. That credential does not exist in this repository today, and the
  issue explicitly says the cost/credential question must be settled before
  code ships — not discovered after.
- The queue picture moves: waiting-ahead counts tick up and down as other
  patients are served, and ADR-0009 lets CarePath recompute plans freely
  (FR-28). A naive "send when near" evaluator would fire repeatedly on the
  same step as the count wobbles across the threshold — the patient blocks
  the official account, and the feature is dead.
- The notification lands on the phone's lock screen. Anything clinical in
  the text (department, doctor, order) leaks PHI to whoever glances at the
  screen.
- ADR-0012 says the server sends stable codes and the web catalogs own the
  patient-facing words. A push message has **no web client in the loop** to
  map a code — the constraint cannot be satisfied by the existing mechanism.

## Decision

### 1. Notifications go through a `Notifier` port; LINE is one adapter, no-op is the default

`internal/notification` defines the port (`Notify(ctx, recipient, text)`).
Two adapters ship:

- **no-op** (default): logs the would-be message. Dev/demo runs, and any
  environment without Messaging API credentials, get the full engine —
  trigger, dedupe, opt-out — with the send itself visibly suppressed.
- **LINE Messaging API**: pushes a text message to the LINE user id via
  `POST /v2/bot/message/push`, selected at startup when
  `LINE_MESSAGING_CHANNEL_TOKEN` is set. The token is the operator's
  opt-in to real (billed) sends.

This is ADR-0004's provider pattern applied to outbound messages: future
channels (SMS, email) are new adapters, not new engines. Until the channel
exists in the operator's LINE Developers console, the adapter is
unit-tested against a scripted fake server and marked as such — enabling it
is an environment change, not a code change.

### 2. "Near" is judged per visit on the recommended step, by a background sweep

A ticker in the API process (default every 30s, `QUEUE_NOTIFY_INTERVAL`)
sweeps active visits and asks the journey module for the queue picture it
already serves (FR-17). The criterion is `waitingAhead <= 3` on the
**recommended** step only — the same single primary action the patient
screen shows, so the notification never contradicts the app. The poll is
server-driven on purpose: the patient who closed the app is exactly the one
who needs the message, so the patient's own screen polls must not be the
trigger.

### 3. Exactly once per step per visit — dedupe is a database row, not a flag

`carepath.queue_notification` keys on `(visit_id, step_key)` with
`ON CONFLICT DO NOTHING`: the sweep first claims the row, and only the
insert that wins sends. Recomputation (FR-28), queue wobble across the
threshold, and API restarts all hit the same primary key — the message
cannot repeat for a step once sent. A visit without a resolvable recipient
does **not** claim the row: a later real login can still be notified.

### 4. Opt-out is per visit and patient-owned

`PUT /api/v1/journeys/{visitId}/notifications` (session + claim guarded,
like every patient-surface write) stores the preference in
`carepath.visit_notification_pref`; absence means enabled. The journey
screen exposes the toggle. Per visit, not per LINE user: the visit is the
unit the patient experiences, and MVP patients have no account page.

### 5. The message text is a fixed constant owned by the notification module — a declared ADR-0012 exception

The push payload carries its own words, so the catalog mechanism cannot
reach it. The notification module therefore owns one fixed, generic Thai
sentence ("queue is approaching — please return to your recommended service
point"). It names no step, clinic, doctor, or order — the no-PHI rule is
absolute and the price of the exception is a single sentence that cannot be
localized per patient. If multilingual push ever matters, the preference
row grows a locale column and the module picks from a fixed set — still no
per-visit clinical data in the text.

## Consequences

- The engine (criterion, dedupe, opt-out, recipient resolution) is fully
  testable and demoable without LINE credentials; production sends need one
  env var plus the operator-side channel. The credential blocker is honored
  by construction, not by deferring the feature.
- `internal/notification` reads the claim table (`patient_visit_claim`)
  through its own recipient port to map visit → LINE user id — demo-mode
  visits (no LINE identity) resolve no recipient and are skipped, keeping
  ADR-0010's identity separation intact.
- Every sweep is O(active visits) queue reads — fine at hackathon scale;
  the port shape leaves room for a smarter index later without touching the
  adapters.
