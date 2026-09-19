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

For a normal hospital floor navigation graph, route calculation should feel interactive. Pre-load/caching of mostly static map graph data is allowed.

## NFR-06 Observability

Services should expose structured logs and health endpoints; OpenTelemetry is recommended for production evolution.

## NFR-07 Contract-first integration

Externally visible HTTP contracts are described with OpenAPI and version controlled.

## NFR-08 Security

- Verify LINE identity/session tokens server-side when production LINE login is enabled.
- Authenticate staff/admin endpoints.
- Keep HIS credentials server-side only.
- Use TLS for production traffic.
