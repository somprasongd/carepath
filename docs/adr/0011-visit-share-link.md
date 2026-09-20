# ADR-0011: Visit Share Link — a Third, Visit-Scoped Read-Only Credential for Relatives

- Status: Accepted
- Date: 2026-09-20
- Related: [ADR-0010](0010-staff-auth-jwt-argon2.md) (identity kinds; §7 open patient surfaces; refresh-token hashing), [ADR-0006](0006-rest-openapi.md) (contract-first), [NFR-03](../requirements/non-functional-requirements.md) (data minimisation), [NFR-08](../requirements/non-functional-requirements.md) (no token in logs), FR-24 / C2

## Context

ADR-0010 fixed two identity kinds and only two: patients arrive from a verified
LINE identity and get an opaque `session` token; staff arrive from
username/password and get a JWT access token plus a rotating refresh token.
The two never cross, and every endpoint guards with exactly one of them.

The relative-tracking story (FR-24, README §6) needs a third thing that is
deliberately *not* an identity. A relative opening a link on their phone:

- is not a person CarePath knows — no LINE identity, no hospital account,
  and nobody should have to log in to see "ซึ่งตอนนี้คุณยายอยู่ขั้นตอนรับยา";
- must see exactly one visit's progress and nothing else;
- must lose access when the patient says so, and automatically after a few
  hours — the link will be forwarded through chat apps and is beyond our
  control the moment it is sent.

Issue #89 spells out the requirements: read-only, time-limited, revocable,
token never in the URL path (the request logger puts `c.Path()` on every log
line — a path token would write a credential into logs, violating NFR-08).

## Decision

Add a **third credential mechanism under explicit conditions**, in a new
`internal/share` module with its own table `carepath.visit_share_link`:

1. **A share link is a permission, not a person.** It is bound to one
   `visit_id`, grants one read (`GET /api/v1/shared/journey`), and nothing
   else. It authenticates nowhere else and nothing else authenticates on that
   route — the contract carries a third security scheme, `shareAuth`,
   precisely so the three token families cannot be silently mixed.

2. **Why not reuse `session`?** A session *stands in for a verified identity*
   and is indistinguishable from the patient to every route that accepts it.
   Reusing it would hand a relative the entire patient surface for that
   identity's lifetime (24 h by default) and would inherit LINE-identity
   semantics (one per person) that have nothing to do with "let my daughter
   see this one visit". Minting a narrower thing is one small module; scoping
   `session` after the fact would touch every patient route.

3. **What the link may reveal** (NFR-03 — data minimisation; `SharedJourney`
   is its own schema and must never reuse `Journey`, so a future field on
   `Journey` cannot silently leak):
   - the current step as human-readable Thai text (e.g. "รอเจาะเลือด"),
   - a coarse status only: `WAITING` / `IN_SERVICE` / `DONE`,
   - the service point's display name and floor,
   - when the view was last updated, and when the link expires.

   Everything else is redacted by construction, not by filtering: patient
   name, `patientRef`/HN/VN, `visitId`, `stepKey` (it embeds the clinic
   code, e.g. `CLINIC:MED:2`), `clinicCode`, `orderRefs`, and the full step
   list never enter the struct. Service-point names carry codes in
   parentheses ("อายุรกรรม (MED)"), so the shared view strips the suffix.

4. **Lifecycle.** `SHARE_LINK_TTL` (default `4h`) sets expiry; `DELETE
   /api/v1/journeys/{visitId}/share` revokes every still-active link of the
   visit ("หยุดแชร์") and is idempotent; at most **5 active links** per
   visit, beyond which creation returns 409 — a brake on token-churn abuse.
   Expired/revoked rows are harmless and stay (the same call ADR-0010 made
   for `refresh_token`); cleanup is out of scope.

5. **Token handling.** 32 random bytes from `crypto/rand`, hex-encoded,
   returned **once** at creation (`{token, expiresAt}` — no full URL, so the
   API never needs to know its own origin); stored as **sha256(token)**,
   exactly like `refresh_token` (fast hash is correct here: the token is
   256-bit random, not a user password). The token travels in
   `Authorization: Bearer` only — never in a path or query — and the web app
   keeps it in the URL **fragment** (`/shared#<token>`), which browsers never
   send to any server. Resolution answers *every* failure — unknown,
   expired, revoked — with the same 401 `unauthorized`, so the endpoint
   cannot be probed for which links existed.

6. **Who may create a link.** `POST .../share` requires a valid patient
   session (`bearerAuth`). Today a session is not bound to a specific visit
   (the LINE user→visit mapping is known outstanding work); requiring *some*
   patient session is still the difference that matters: an outsider who
   guesses a `visitId` cannot mint a hours-long credential from it, they
   would already need a live patient session. `DELETE` requires the same.

7. **Honest risk statement.** `GET /api/v1/journeys/{visitId}` remains open
   by design (ADR-0010 §7 — the patient screens have no login). The share
   link therefore does not *worsen* exposure: anyone who guesses a visit id
   can already read the full journey. What the link adds is a *narrower*
   view for a *wider* audience, plus creation gated on a patient session.
   Binding journey reads to the owning session is separate, older work and
   is not smuggled into this ADR. *(Update 2026-09-20: that work landed as
   #96 — the journey read now takes the patient session plus a visit claim,
   so the risk stated here is closed.)*

## Consequences

- Three token kinds now exist. The rule generalises: **a credential is valid
  only on the surfaces its own scheme names**, in every direction — cross-use
  is tested in both directions (`internal/e2e/share_test.go`).
- `docs/README.md`, root `AGENTS.md` boundary 9, and the OpenAPI contract
  (v0.12.0, `shareAuth`) all name the third mechanism, so the next public
  endpoint decides consciously which scheme — if any — it accepts.
- Relatives get no map, no route, no notifications (S6) — a link shows the
  step, the coarse status, and where in the building that is by name/floor.
- **Pre-deployment checklist** (compose ships demo-friendly defaults):
  1. set `ALLOW_DEMO_AUTH=false` (`.env.example` now defaults `true` because
     compose *is* the demo machine — flip it for any real deployment),
  2. set a real `JWT_SECRET` (ADR-0010),
  3. review `SHARE_LINK_TTL` against hospital policy,
  4. serve over HTTPS — the token rides a header, and the web fragment must
     not leak through a plain-HTTP hop.

## Out of scope

- All screens — the web side (share button, `/shared#token` page) is #90.
- Map/navigation views and queue notifications for relatives (S6).
- Periodic purging of expired rows (harmless; revisit if the table grows).
- Binding `GET /api/v1/journeys/{visitId}` to the owning patient session
  (landed since, as #96).
