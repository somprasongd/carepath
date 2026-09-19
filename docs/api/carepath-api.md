# CarePath API - Initial Contract

Base path: `/api/v1`

Source of truth for externally visible behavior: [`packages/contracts/openapi/carepath.yaml`](../../packages/contracts/openapi/carepath.yaml) (ADR-0006). The running API also serves a swaggo-generated spec at `GET /api/openapi.json` and a Swagger UI at `GET /swagger` — regenerate the generated docs with `make swag`.

## Health

`GET /health`

## Journey

`GET /api/v1/journeys/{visitId}`

The journey plan CarePath derived from HIS facts ([ADR-0009](../adr/0009-carepath-owns-journey-plan.md)) — the HIS has no concept of an ordered journey, so CarePath decides which steps exist and their order. Each step carries a stable `stepKey` (its identity across replans) alongside a `sequence` (display order only). `actionable` lists every step currently `READY`; `recommended` is CarePath's pick among them for the patient's single primary action. 404 while the visit has not been ingested yet.

`POST /api/v1/journeys/{visitId}/steps/{stepKey}/transition`

Staff command to move a step to `STARTED`, `COMPLETED`, or `CANCELLED`. CarePath owns step status directly (ADR-0009) — this no longer forwards anything to the HIS. Returns the refreshed journey with the plan recomputed against the new fact.

`POST /api/v1/journeys/{visitId}/clinics/{clinicCode}/close-round`

Staff override: confirms a clinic is done with the patient for this round even without an `encounter.completed` fact from the HIS, dropping any not-yet-started "return to this clinic" step the planner had inferred.

## Staff visit monitor

`GET /api/v1/staff/visits`

The projected journey of every visit CarePath knows, freshest sync first — same per-visit shape as the single-journey read (#37).

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

`POST /api/v1/locations/resolve`

Normalizes provider input such as QR into a CarePath location result.

## Removed

`GET /api/v1/visits/{visitId}` and `GET /api/v1/visits/{visitId}/next` (the
`visit` module) predated the journey planner and assumed the HIS reports an
ordered step list, which it does not (ADR-0009). They were deprecated when the
contract first moved to the planner model and removed once nothing depended on
them any longer — superseded fully by `GET /api/v1/journeys/{visitId}`.

Implemented today: health, auth session, the journey projection with step
transitions and the clinic round override, the staff visit monitor, and the
service point reads (with places/floors resolved from Postgres). The route
and location endpoints remain intentionally documented as the next build
slices.
