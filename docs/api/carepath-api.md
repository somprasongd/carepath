# CarePath API - Initial Contract

Base path: `/api/v1`

Source of truth for externally visible behavior: [`packages/contracts/openapi/carepath.yaml`](../../packages/contracts/openapi/carepath.yaml) (ADR-0006). The running API also serves a swaggo-generated spec at `GET /api/openapi.json` and a Swagger UI at `GET /swagger` — regenerate the generated docs with `make swag`.

## Health

`GET /health`

## Authentication

Two credentials, two audiences — see [ADR-0010](../adr/0010-staff-auth-jwt-argon2.md).

| | Patient | Staff / admin |
|---|---|---|
| Login | `POST /api/v1/auth/session` (LINE ID token, or the demo bypass) | `POST /api/v1/auth/login` (username + password) |
| Credential | opaque session token (`bearerAuth`) | JWT access token (`staffAuth`) + opaque refresh token |
| Lifetime | 24h (`SESSION_TTL`) | 15 min access (`ACCESS_TOKEN_TTL`), 7 day refresh (`REFRESH_TOKEN_TTL`) |
| Who am I | `GET /api/v1/auth/session` | `GET /api/v1/auth/me` |

`POST /api/v1/auth/login`

`{username, password}` → access token, refresh token, both expiries, and the
identity with its role codes. Passwords are argon2id hashes, verified
server-side. An unknown username, a wrong password, and a deactivated account
return the same 401 — no user enumeration.

`POST /api/v1/auth/refresh`

`{refreshToken}` → a new pair. Refresh tokens are single-use: this call spends
the presented one. Replaying a spent token revokes every refresh token of that
user and returns 401.

`POST /api/v1/auth/logout`

`{refreshToken}` → 204. Idempotent — an unknown or already-revoked token also
returns 204. The access token is stateless and stays valid until it expires.

`GET /api/v1/auth/me`

The staff identity and roles behind the presented access token.

### Which endpoints require a staff token

Send `Authorization: Bearer <accessToken>`; a missing or expired token is 401,
a valid token without a permitted role is 403.

| Endpoint | Roles |
|---|---|
| `GET /api/v1/staff/visits` | `STAFF`, `ADMIN` |
| `POST /api/v1/journeys/{visitId}/steps/{stepKey}/transition` | `STAFF`, `ADMIN` |
| `POST /api/v1/journeys/{visitId}/clinics/{clinicCode}/close-round` | `STAFF`, `ADMIN` |
| `GET /api/v1/analytics/overview` | `STAFF`, `ADMIN`, `EXECUTIVE` |
| `GET /api/v1/auth/me` | any authenticated staff user |

The `EXECUTIVE` role (#86, seeded as `exec`/`demo`) exists for exactly one
surface: the analytics overview. An executive token on any other staff
endpoint is 403 — least privilege, so a dashboard login never doubles as an
operations login.

Everything else — the journey read, service points, location, and route — is
open in the MVP, because the patient screens consume it. Binding the journey
read to the patient's own session is tracked separately (ADR-0010 §7).

## Journey

`GET /api/v1/journeys/{visitId}`

The journey plan CarePath derived from HIS facts ([ADR-0009](../adr/0009-carepath-owns-journey-plan.md)) — the HIS has no concept of an ordered journey, so CarePath decides which steps exist and their order. Each step carries a stable `stepKey` (its identity across replans) alongside a `sequence` (display order only). `actionable` lists every step currently `READY`; `recommended` is CarePath's pick among them for the patient's single primary action. 404 while the visit has not been ingested yet.

`POST /api/v1/journeys/{visitId}/steps/{stepKey}/transition`

Staff command to move a step to `STARTED`, `COMPLETED`, or `CANCELLED`. CarePath owns step status directly (ADR-0009) — this no longer forwards anything to the HIS. Returns the refreshed journey with the plan recomputed against the new fact. Requires a `STAFF` or `ADMIN` access token; the authenticated user is recorded as the actor in the command audit (NFR-09).

`POST /api/v1/journeys/{visitId}/clinics/{clinicCode}/close-round`

Staff override: confirms a clinic is done with the patient for this round even without an `encounter.completed` fact from the HIS, dropping any not-yet-started "return to this clinic" step the planner had inferred. Requires a `STAFF` or `ADMIN` access token.

## Staff visit monitor

`GET /api/v1/staff/visits`

The projected journey of every visit CarePath knows, freshest sync first — same per-visit shape as the single-journey read (#37). Requires a `STAFF` or `ADMIN` access token; the `/api/v1/staff` prefix exists so one middleware covers the whole group.

## Analytics

`GET /api/v1/analytics/overview?window=today`

Executive dashboard numbers (#86, epic #83), aggregated in SQL from the
append-only step timeline (#85): per service point — currently waiting /
in-progress counts, longest current wait, average wait, average service
time, completed today; visit-wide — active visits, average visit length,
average wait, and the bottleneck service point (highest average wait, ties
by who is still waiting). Only `window=today` exists; anything else is 400.
"Today" is midnight-to-now in the analytics timezone
(`ANALYTICS_TIMEZONE`, default `Asia/Bangkok`) — resolved by the database,
not the API process's clock. Averages with no samples are `null`, never 0:
no-data and zero-minutes are different facts. Requires a `STAFF`, `ADMIN`,
or `EXECUTIVE` token; never returns patient-level data (NFR-03).

## Service points

`GET /api/v1/service-points`

`GET /api/v1/service-points/{code}`

Returns the active service points (or one by binding code — a clinic code
`CLINIC:MED`, an order type `ORDERTYPE:LAB`, or a fixed code like `CASHIER`)
with the resolved `place` and nested `floor` — the step → destination mapping
(#24, extended by ADR-0009). Read-only; mapping changes go through seed
migrations for the MVP.

## Route

`GET /api/v1/navigation/route?from={nodeId}&to={nodeId}`

Returns route node IDs/coordinates for overlay on the floor plan.

## Location

`GET /api/v1/journeys/{visitId}/location`

The visit's current location — the latest recorded observation. Canonical start point for routing. 404 if nothing has been recorded yet.

`POST /api/v1/journeys/{visitId}/location`

Reports a scanned location fix (QR today) and normalizes it via the source's provider (ADR-0004) into a canonical navigation node, recorded as the current location.

`POST /api/v1/demo/zigbee/location`

Demo-only Zigbee simulator (#33): reports a zone-level fix through the `ZIGBEE` location provider, standing in for a real Zigbee integration during demos.

## Removed

`GET /api/v1/visits/{visitId}` and `GET /api/v1/visits/{visitId}/next` (the
`visit` module) predated the journey planner and assumed the HIS reports an
ordered step list, which it does not (ADR-0009). They were deprecated when the
contract first moved to the planner model and removed once nothing depended on
them any longer — superseded fully by `GET /api/v1/journeys/{visitId}`.

Implemented today: health, auth session (patient), staff authentication
(`login`/`refresh`/`logout`/`me` per ADR-0010 — the staff visit monitor, step
transitions, and the clinic round override now require a `STAFF`/`ADMIN`
access token), the journey projection, the service point reads (with places/floors
resolved from Postgres), the navigation route API, and the location
report/read endpoints (QR + Zigbee simulator).
