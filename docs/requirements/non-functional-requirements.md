# Non-Functional Requirements

## NFR-01 Maintainability

Care Graph, hospital spatial model, routing, and external HIS integration are separate modules with explicit interfaces.

## NFR-02 Replaceability

Mock HIS can be replaced by a real HIS adapter without changing CarePath domain logic.

## NFR-03 Privacy

Do not expose unnecessary patient information in URLs, QR payloads, logs, or floor-plan assets. Use opaque visit/session identifiers where practical. As of ADR-0009, CarePath stores one piece of PHI — the patient's display name, for staff-facing screens only — nothing else; it must never appear on a patient-facing endpoint beyond the patient's own visit, in a URL, or in a QR payload.

## NFR-04 Availability degradation

If realtime positioning is unavailable, navigation must still be usable with QR/manual location.

## NFR-05 Performance

For a normal hospital floor navigation graph, route calculation should feel interactive. Pre-load/caching of mostly static map graph data is allowed. Primary patient-facing screens must render within 3 seconds against the prepared demo dataset.

## NFR-06 Observability

Services should expose structured logs and health endpoints; OpenTelemetry is recommended for production evolution.

## NFR-07 Contract-first integration

Externally visible HTTP contracts are described with OpenAPI and version controlled.

## NFR-08 Security

- Verify LINE identity/session tokens server-side when production LINE login is enabled.
- Authenticate staff/admin endpoints and enforce role-based authorization (see FR-18).
- Store passwords one-way hashed; never store or log plaintext passwords. Concretely, per [ADR-0010](../adr/0010-staff-auth-jwt-argon2.md): **argon2id** (m=64 MiB, t=3, p=2, 16-byte per-user salt, 32-byte key), stored as a PHC string so the parameters travel with the hash, verified in constant time.
- Authentication failures must not leak which half was wrong: an unknown username, a wrong password, and a deactivated account return the same response and take the same time.
- Access tokens are short-lived (15 min default) and signed server-side; refresh tokens are opaque, stored only as a hash, single-use with rotation, and revoked wholesale for a user when a spent one is replayed.
- Never log a token, a password, or an `Authorization` header value — not even truncated.
- Demo credentials (`admin`/`demo`, `staff`/`demo`) are for local development and the hackathon demo only. Any deployment reachable by others must first run the production checklist in ADR-0010 §12: a real `JWT_SECRET` from a secret store, seed users removed or given new passwords, TLS on.
- Use parameterized queries / prepared statements for all database access to prevent SQL injection.
- Keep HIS credentials server-side only.
- Use TLS for production traffic.

## NFR-09 Audit trail

Every status change on a visit, visit step, or queue ticket records who made the change, when (system-generated timestamp), and the previous and new status.

"Who" means the authenticated user behind the request, not just the surface it came from: `carepath.journey_command_audit` carries both — `source` for the surface (`staff-web`) and the actor's user id/username from the access token ([ADR-0010](../adr/0010-staff-auth-jwt-argon2.md) §11). Both actor columns are nullable, because system-initiated writes (the HIS ingest poller) legitimately have no user.

## NFR-10 Usability

A user with no IT background can operate the system without training. Error messages are human-readable; raw system errors or error codes are never shown to end users.

## NFR-11 Accessibility

Patient-facing screens support large-text display, sufficient color contrast for readability, and must be usable on small (mobile) screens.

## NFR-12 Backup and recovery

A backup and recovery approach for the database must be documented (design-level for the hackathon; an operational implementation is not required).
