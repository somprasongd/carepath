# Non-Functional Requirements

## NFR-01 Maintainability

Care Graph, hospital spatial model, routing, and external HIS integration are separate modules with explicit interfaces.

## NFR-02 Replaceability

Mock HIS can be replaced by a real HIS adapter without changing CarePath domain logic.

## NFR-03 Privacy

Do not expose unnecessary patient information in URLs, QR payloads, logs, or floor-plan assets. Use opaque visit/session identifiers where practical.

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
- Store passwords one-way hashed; never store or log plaintext passwords.
- Use parameterized queries / prepared statements for all database access to prevent SQL injection.
- Keep HIS credentials server-side only.
- Use TLS for production traffic.

## NFR-09 Audit trail

Every status change on a visit, visit step, or queue ticket records who made the change, when (system-generated timestamp), and the previous and new status.

## NFR-10 Usability

A user with no IT background can operate the system without training. Error messages are human-readable; raw system errors or error codes are never shown to end users.

## NFR-11 Accessibility

Patient-facing screens support large-text display, sufficient color contrast for readability, and must be usable on small (mobile) screens.

## NFR-12 Backup and recovery

A backup and recovery approach for the database must be documented (design-level for the hackathon; an operational implementation is not required).
