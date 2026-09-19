# ADR-0010: Staff/Admin Authentication with argon2id Passwords and JWT Access/Refresh Tokens

- Status: Accepted
- Date: 2026-09-20
- Supersedes: the shared-token approach originally proposed in issue #43
- Related: [ADR-0006](0006-rest-openapi.md) (contract-first), [ADR-0007](0007-hexagonal-modules-transaction-in-context.md) (module shape), [FR-18](../requirements/functional-requirements.md), [NFR-08](../requirements/non-functional-requirements.md), [NFR-09](../requirements/non-functional-requirements.md)

## Context

Every staff/admin endpoint CarePath serves today is open to anyone who can
reach port 8080: the staff visit monitor (`GET /api/v1/staff/visits`), the step
transition command, and the clinic-round override. The web console's `/login`
screen exists but is inert — it takes a username and password, ignores both,
and lets the caller pick their own role from four cards. `M8` /
[FR-18](../requirements/functional-requirements.md) ("authentication and
role-based access control") is the last Must-Have with nothing behind it.

The patient side is already solved differently: `internal/session` exchanges a
LINE ID token (verified against LINE's JWKS) for an opaque CarePath session
token stored in `carepath.patient_session`. That mechanism does not transfer to
staff — a nurse at a service point has no LINE identity to present, and the
hospital expects a username and password issued by its own IT.

Issue #43 proposed the smallest possible thing: one shared bearer token read
from an environment variable, checked by a middleware. That was rejected here:

- **It cannot answer "who".** [NFR-09](../requirements/non-functional-requirements.md)
  requires every status change to record who made it. A token shared by every
  workstation records nothing; `journey_command_audit.source = "staff-web"` is
  a surface, not a person.
- **It cannot express a role.** FR-18 is explicitly *role-based* access
  control. One token has one privilege level, so admin-only surfaces (map and
  service-point administration, user management) have nowhere to land.
- **It cannot be revoked for one person** — rotating it logs out the whole
  hospital.
- **It is thrown away on first contact with a real deployment**, and the work
  to replace it is exactly the work below. The login screen already exists; the
  ER diagram in [deliverable 02](../deliverables/02-er-diagram-database.md) §E
  already designs `app_user` / `role` / `user_role`. What is missing is the
  module, the migration, and the contract — not a research spike.

## Decision

### 1. A new `auth` module, kept separate from `session`

`internal/auth` owns staff/admin users, credential verification, and token
issue/refresh/revocation. It does not absorb, and is not absorbed by, the
patient `session` module.

| | `session` (existing) | `auth` (new) |
|---|---|---|
| Subject | patient / relative | staff, admin |
| Identity source | LINE ID token (JWKS), or the demo bypass | username + password CarePath itself stores |
| Credential issued | opaque session token | JWT access token + opaque refresh token |
| Lifetime | 24h fixed (`SESSION_TTL`) | 15 min access, 7 day rotating refresh |
| Revocation | expiry only | logout, rotation, reuse detection |
| Tables | `carepath.patient_session` | `carepath.app_user`, `role`, `user_role`, `refresh_token` |

Merging the two was considered and rejected: a patient never has a password and
a staff member never has a LINE identity, so one table would be two disjoint
halves behind a discriminator column, with different lifetimes and different
revocation rules bolted on top. Keeping them apart also keeps the blast radius
of a change to either one small. Both remain ordinary ADR-0007 modules — other
modules may call `auth.Service`, never its `Repo`.

### 2. Passwords: argon2id, encoded in PHC string format

Hashing uses `golang.org/x/crypto/argon2` (`argon2.IDKey`) — already an
indirect dependency of `apps/api`, promoted to a direct one. Chosen over bcrypt
because argon2id is the OWASP first-choice password hash and is memory-hard;
there is no legacy hash corpus to stay compatible with.

Parameters (OWASP baseline for argon2id, comfortably affordable for the
handful of logins this system sees):

| Parameter | Value |
|---|---|
| memory | 64 MiB (`m=65536`) |
| iterations | 3 (`t=3`) |
| parallelism | 2 (`p=2`) |
| salt | 16 random bytes, per user, from `crypto/rand` |
| key length | 32 bytes |

The stored value is the standard PHC string, so the parameters travel with the
hash and can be raised later without a flag day:

```text
$argon2id$v=19$m=65536,t=3,p=2$<base64 salt>$<base64 hash>
```

Verification re-derives with the parameters parsed out of the stored string and
compares with `crypto/subtle.ConstantTimeCompare`.

Two rules the implementation must keep:

- **A failed login never says which half was wrong.** Unknown username and bad
  password both return the same 401 and the same message
  (`"ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง"` at the UI, `"invalid credentials"` in the
  API error body) — no user enumeration.
- **An unknown username still runs one argon2 verify** against a fixed dummy
  hash before returning, so response time does not reveal whether the account
  exists.

### 3. Access token: a stateless HS256 JWT, short-lived

Signed with `github.com/golang-jwt/jwt/v5` (already a direct dependency, used
for LINE token verification). Default TTL 15 minutes.

| Claim | Value |
|---|---|
| `iss` | `carepath-api` |
| `sub` | `app_user.user_id` |
| `username` | the login name, for logs and the UI header |
| `name` | display name |
| `roles` | array of role codes, e.g. `["ADMIN"]` |
| `typ` | `access` — a refresh token is never accepted as an access token |
| `iat`, `exp`, `jti` | standard |

Verification is a signature + claims check with no database round trip, which
is the point: the staff console polls `GET /api/v1/staff/visits` and must not
pay a query per request to authenticate. The cost of statelessness is that a
revoked user keeps a working access token until it expires — bounded at 15
minutes, and acceptable at this scale.

### 4. Refresh token: opaque, hashed at rest, single-use, revocable

Not a JWT. A refresh token must be revocable, which means a server-side record
exists anyway; making it a JWT would only add a second long-lived bearer
credential that cannot be withdrawn.

- 32 random bytes from `crypto/rand`, hex-encoded, handed to the client once.
- Stored as `sha256(token)` in `carepath.refresh_token` — a database leak does
  not yield usable tokens. (No argon2 here: the token is already 256 bits of
  entropy, so it is not brute-forceable and does not need a slow hash.)
- Default TTL 7 days (`REFRESH_TOKEN_TTL`).
- **Single use with rotation:** `POST /api/v1/auth/refresh` marks the presented
  token used and issues a new access + refresh pair.
- **Reuse detection:** presenting an already-used token means it leaked. Every
  refresh token for that user is revoked and the call returns 401 — the real
  user is forced to log in again, and the thief gains nothing.
- Logout revokes the presented refresh token.

### 5. Roles: `ADMIN` and `STAFF` for the MVP

The role model stays the many-to-many shape already designed in
[deliverable 02](../deliverables/02-er-diagram-database.md) §E (`app_user` ×
`role` via `user_role`), but the MVP seeds and enforces exactly two role codes:

| Role | Who | Can |
|---|---|---|
| `STAFF` | registration/screening and service-point staff | read the staff visit monitor, transition step status, close a clinic round |
| `ADMIN` | hospital IT / system administrator | everything `STAFF` can, plus admin-only surfaces as they land (map and service-point administration, user management) |

The four role cards on the login screen (registration, service-point, admin,
executive) are **removed**: a user does not choose their own privileges. Role
comes from the token, and the console lands the user on the screen their role
implies. Splitting `STAFF` into `REGISTRATION_STAFF` / `SERVICE_POINT_STAFF`,
or adding `EXECUTIVE` when the S7 dashboard is built, is then a data change
(two inserts) rather than a schema or contract change.

### 6. Endpoints

Contract-first per [ADR-0006](0006-rest-openapi.md) — the shapes live in
`packages/contracts/openapi/carepath.yaml`.

| Endpoint | Auth | Purpose |
|---|---|---|
| `POST /api/v1/auth/login` | none | `{username, password}` → access token, refresh token, expiry, identity |
| `POST /api/v1/auth/refresh` | none (the refresh token is the credential) | `{refreshToken}` → a new pair; the presented token is spent |
| `POST /api/v1/auth/logout` | none (idempotent) | `{refreshToken}` → 204; unknown/expired tokens also return 204 |
| `GET /api/v1/auth/me` | staff access token | the identity and roles behind the presented token |

`POST /api/v1/auth/session` (patient, LINE) is untouched and keeps its own
`bearerAuth` scheme. The contract gains a second security scheme,
`staffAuth`, so a reader can tell which credential an endpoint expects.

### 7. What is protected, and what deliberately is not

| Endpoint | MVP requirement |
|---|---|
| `GET /api/v1/staff/visits` | `STAFF` or `ADMIN` |
| `POST /api/v1/journeys/{visitId}/steps/{stepKey}/transition` | `STAFF` or `ADMIN` |
| `POST /api/v1/journeys/{visitId}/clinics/{clinicCode}/close-round` | `STAFF` or `ADMIN` |
| `GET /api/v1/auth/me` | any authenticated staff user |
| `GET /api/v1/journeys/{visitId}` | **unchanged — open** |
| `GET /api/v1/service-points`, `/{code}` | **unchanged — open** (read-only reference data) |
| `GET|POST /api/v1/journeys/{visitId}/location`, `/api/v1/navigation/route` | **unchanged — open** (patient surfaces) |
| `POST /api/v1/auth/session` | unchanged (patient login itself) |

`GET /api/v1/journeys/{visitId}` stays open on purpose: it is the patient
screens' only data source, and the staff detail view reads the same endpoint.
FR-18's second half — "a patient can see only their own visit data" — means
binding that read to the patient's own session, which is a separate slice of
work on the `session` side and is **not** in this ADR. Until it lands, knowing
a visit id is enough to read that journey; that is a documented gap, not an
oversight.

A middleware that silently protected the journey read would break every
patient screen, so protection is applied per route group in the composition
root, never globally.

### 8. Middleware

```go
// internal/auth/middleware.go
func RequireRole(s Service, roles ...string) fiber.Handler
```

It extracts the `Authorization: Bearer <jwt>`, verifies signature/expiry/`typ`,
rejects with the module's `apperr` values, and puts the resolved principal in
the request context (`auth.PrincipalFromContext(ctx)`) so handlers and the
audit trail can name the actor. Errors follow the existing model
(`apps/api/AGENTS.md`): `apperr.KindUnauthorized` → 401 for a missing, invalid,
or expired token; `apperr.KindForbidden` → 403 for a valid token whose roles do
not include any of the required ones. Both are logged once at the boundary by
`httpx.Error`.

### 9. Seeded demo accounts

A seed migration creates the two roles and two users, following the pattern
already used by `000002_seed_service_points` and `000004_seed_xray_service_point`:

| Username | Password | Role |
|---|---|---|
| `admin` | `demo` | `ADMIN` |
| `staff` | `demo` | `STAFF` |

These are **demo credentials in a public repository**: the argon2id hash of
`demo` is committed in the migration, so it must be treated as known to
everyone. Recorded consequences:

- The seed users are for the hackathon demo and local development only.
- Any deployment reachable by anyone else must run the production checklist in
  §12 before it is exposed.
- The seed rows are `ON CONFLICT DO NOTHING`, so changing a password in a live
  database is not undone by re-running migrations.

### 10. Configuration

| Variable | Default | Meaning |
|---|---|---|
| `JWT_SECRET` | *(unset)* | HS256 signing key. **Unset → the server generates a random 32-byte key at boot and logs a warning**; tokens then stop working across a restart. There is deliberately no hardcoded default secret, in the same spirit as the existing `LINE_CHANNEL_ID` warning. |
| `ACCESS_TOKEN_TTL` | `15m` | Go duration, parsed with the existing `envDuration` helper |
| `REFRESH_TOKEN_TTL` | `168h` | 7 days |

These join `.env.example` and the `api` service in `docker-compose.yml` when
the module is implemented.

### 11. Audit trail (NFR-09)

`carepath.journey_command_audit` gains `actor_user_id` and `actor_username`,
filled from the authenticated principal. The existing `source` column keeps its
meaning — *which surface* the command came from (`staff-web`) — and the new
columns answer *who*. They are nullable so the ingest poller and other
system-initiated writes stay representable, matching the `user_id text REFERENCES
… -- null = system-initiated` convention in the deliverable-02 design.

### 12. Web client (`apps/web`)

- **Access token in memory only** (React context state), **refresh token in
  `localStorage`**. The access token is the one attached to every request, so
  keeping it out of persistent storage shrinks the XSS payoff; the refresh
  token in `localStorage` is what makes a page reload survive without a second
  login. An httpOnly refresh cookie was considered and rejected for the MVP: it
  forces credentialed CORS, a fixed origin allowlist, and SameSite handling
  across the `5173 → 8080` dev split, for a hackathon demo that is not
  internet-exposed. The trade-off is recorded here rather than hidden.
- `api/client.ts` gains an auth-aware request path: attach the access token,
  and on a 401 attempt exactly one refresh, retry the original request once,
  and on failure clear both tokens and route to `/login`. One refresh in
  flight at a time — concurrent 401s wait on the same promise.
- **The existing single token slot has to become surface-aware first.** `#62`
  added one module-level `authToken` plus `setApiAuthToken`, set by the
  patient auth providers. Adding staff tokens to that same slot would let a
  patient session token be sent to a staff endpoint (and the reverse) in one
  SPA where both route trees are mounted. Keep one slot per audience — the
  patient session token and the staff access token are not interchangeable
  credentials (§1), and the client must not treat them as one.
- A `StaffAuthProvider` (sibling to the patient `AuthProvider`, not a
  replacement) holds the principal; `/staff/*` mounts behind a
  `RequireStaffAuth` guard the way `/patient/*` mounts behind `LoginGate`.
- `/login` loses the role cards and gains real form submission, a pending
  state, and an inline error. Landing screen is derived from the role in the
  token.

## Schema

```sql
CREATE TABLE IF NOT EXISTS carepath.app_user (
    user_id       text PRIMARY KEY,
    username      text NOT NULL UNIQUE,
    password_hash text NOT NULL,          -- argon2id PHC string; never plaintext (NFR-08)
    full_name     text NOT NULL,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS carepath.role (
    role_id text PRIMARY KEY,
    code    text NOT NULL UNIQUE,         -- MVP: 'ADMIN', 'STAFF'
    name    text NOT NULL
);

CREATE TABLE IF NOT EXISTS carepath.user_role (
    user_id text NOT NULL REFERENCES carepath.app_user (user_id) ON DELETE CASCADE,
    role_id text NOT NULL REFERENCES carepath.role (role_id),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS carepath.refresh_token (
    token_hash text PRIMARY KEY,          -- sha256 of the opaque token, never the token
    user_id    text NOT NULL REFERENCES carepath.app_user (user_id) ON DELETE CASCADE,
    issued_at  timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,               -- non-null = spent by a rotation
    revoked_at timestamptz                -- non-null = logged out or revoked by reuse detection
);

CREATE INDEX IF NOT EXISTS refresh_token_user_idx ON carepath.refresh_token (user_id);
CREATE INDEX IF NOT EXISTS refresh_token_expires_at_idx ON carepath.refresh_token (expires_at);
```

A token is usable only while `used_at IS NULL AND revoked_at IS NULL AND
expires_at > now()`. Expired rows are harmless; a cleanup job is not MVP work.

## Consequences

### Positive

- M8 / FR-18 stops being a mock. The login screen, the ER design, and the
  requirement finally describe the same system.
- NFR-09's "who" becomes real: a step transition records the user that made it.
- NFR-08's "store passwords one-way hashed" is satisfied with the current
  best-practice algorithm rather than a promise in a document.
- Adding a role (`EXECUTIVE` for the S7 dashboard) or splitting `STAFF` is two
  inserts plus a route annotation.
- The patient experience is untouched — no LINE flow, no patient screen, and no
  patient endpoint changes behavior.

### Trade-offs

- A revoked or deactivated user keeps a valid access token for up to 15
  minutes. Accepted; the alternative is a DB lookup per request.
- `HS256` with a shared secret means the API is both issuer and verifier. Fine
  for a monolith; a split into services later would want asymmetric signing
  (RS256/EdDSA) and a JWKS endpoint of CarePath's own.
- Demo credentials are public. Mitigated by §9 and the checklist below, not by
  obscurity.
- Two auth mechanisms now exist in one API. The contract labels them
  (`bearerAuth` vs `staffAuth`) and the module table above names the split, but
  a reader must notice it.

### Explicitly out of scope for this ADR

- Binding `GET /api/v1/journeys/{visitId}` to the patient's own session (the
  second half of FR-18).
- User-management screens (create/disable a user, reset a password) — US-17's
  full scope. MVP manages users through the database.
- Login rate limiting / lockout, password policy, password change and reset,
  MFA, SSO / hospital Active Directory.
- An `EXECUTIVE` role, which has no protected surface until S7 exists.

### Production checklist (before any non-demo deployment)

1. Set a real `JWT_SECRET` (≥32 random bytes) from a secret store, not `.env`.
2. Delete or deactivate the `admin` / `staff` seed users, or change both
   passwords.
3. Serve over TLS (NFR-08) — a bearer token on plain HTTP is a token in transit.
4. Reconsider the access-token TTL and add rate limiting on `/auth/login`.

## Build order

The slices are independent enough to review one at a time, and each one leaves
the tree green:

1. `infra/postgres/migrations/000011_staff_auth.{up,down}.sql` — the four
   tables, the two roles, the two seed users.
2. `infra/postgres/migrations/000012_journey_audit_actor.{up,down}.sql` —
   `actor_user_id` / `actor_username` on `journey_command_audit`.
3. `internal/auth` — `auth.go` (domain + `Repo` port + `apperr` values),
   `password.go` (argon2id hash/verify), `token.go` (JWT issue/verify),
   `service.go`, `handler.go`, `middleware.go`, `postgres/repo.go`.
4. `packages/contracts/openapi/carepath.yaml` + `npm run gen:api` — already
   done in this change; handlers must match it, not drift from it.
5. `cmd/server/main.go` — wire the module and apply `RequireRole` to the staff
   route groups. `make swag`.
6. `apps/web` — `features/auth` (queries + token store), client refresh
   handling, `StaffAuthProvider` + `RequireStaffAuth`, real `/login`.

## Verification

Minimum tests for the slices above:

- argon2id: hash → verify round-trip; a tampered PHC string fails; two hashes
  of the same password differ (per-user salt).
- Token: a valid access token verifies; expired, wrong-signature, and
  `typ=refresh` tokens are rejected.
- Refresh: rotation issues a new pair and spends the old one; presenting a
  spent token returns 401 **and** revokes the user's other tokens; logout makes
  the token unusable.
- Login: correct credentials → 200; wrong password, unknown user, and an
  inactive user → 401 with the identical body.
- Middleware: no header, malformed header, expired token → 401; a `STAFF`
  token on an `ADMIN`-only route → 403; a `STAFF` token on a staff route →
  200; a patient endpoint with no token → still 200 (no regression).
