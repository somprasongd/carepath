# CarePath API - Initial Contract

Base path: `/api/v1`

Source of truth for externally visible behavior: [`packages/contracts/openapi/carepath.yaml`](../../packages/contracts/openapi/carepath.yaml) (ADR-0006). The running API also serves a swaggo-generated spec at `GET /api/openapi.json` and a Swagger UI at `GET /swagger` — regenerate the generated docs with `make swag`.

## Health

`GET /health`

## Visit

`GET /api/v1/visits/{visitId}`

Returns a normalized CarePath view of the visit and its steps.

## Next destination

`GET /api/v1/visits/{visitId}/next`

Returns the next actionable visit step and its service point/place mapping.

## Service points

`GET /api/v1/service-points`

`GET /api/v1/service-points/{code}`

Returns the active service points (or one by service code, e.g. `LAB`) with
the resolved `place` and nested `floor` — the service → destination mapping
(#24). Read-only; mapping changes go through seed migrations for the MVP.

## Route

`GET /api/v1/navigation/route?from={nodeId}&to={nodeId}`

Returns route node IDs/coordinates for overlay on the floor plan.

## Location

`POST /api/v1/locations/resolve`

Normalizes provider input such as QR into a CarePath location result.

Implemented today: health, visit view, next destination, auth session, the
journey projection with step transitions, and the service point reads (with
places/floors resolved from Postgres). The route and location endpoints
remain intentionally documented as the next build slices.
