# Mock HIS

## Purpose

Mock HIS is a deterministic fake upstream hospital system used for the hackathon and automated development. It is not a second implementation of CarePath business logic.

## Responsibilities

Mock HIS provides upstream facts such as:

- patient/visit reference
- visit status
- ordered/required service sequence
- service codes
- simple queue/status data where needed for demonstration

It does **not** calculate indoor routes or own CarePath floor plans.

## Demo visit

The starter project contains one sample visit:

```text
VISIT-001
1. Registration   COMPLETED
2. Screening      COMPLETED
3. Doctor          COMPLETED
4. Laboratory     READY
5. Pharmacy       PENDING
6. Complete       PENDING
```

The CarePath API fetches this upstream shape and converts service codes such as `LAB` into internal service points such as `LAB-01`.

## Integration contract (ADR-0008)

The canonical HIS↔CarePath contract lives in [`packages/contracts/openapi/mock-his.yaml`](../../packages/contracts/openapi/mock-his.yaml) (source of truth per ADR-0006) and has three surfaces:

- **Snapshot read (pull):** `GET /api/v1/visits/{visitId}` — what CarePath renders the journey from.
- **Transition command (push to HIS):** `POST /api/v1/visits/{visitId}/steps/{sequence}/transition` with `{commandId, to}` where `to` is `STARTED | COMPLETED | CANCELLED`. This is the only way step status changes; the HIS is the system of record, CarePath only projects.
- **Event feed (pull):** `GET /api/v1/events?after={eventId}` — append-only canonical events (`visit.opened/updated`, `service.requested/started/completed/cancelled`) for audit/replay and a future push transport.

Rules the contract encodes:

- **Idempotency:** events carry an HIS-assigned `eventId`; commands carry a caller-assigned `commandId`. Replaying a command, or transitioning to the step's current status, is a no-op success.
- **Canonical enums:** visit status `ACTIVE | COMPLETED | CANCELLED`, step status `PENDING | READY | STARTED | COMPLETED | CANCELLED`. A real HIS adapter translates vendor codes into these.
- **External IDs only:** payloads carry `visitId` / `patientRef` (opaque, no PHI) and `serviceCode` — never CarePath-side ids or coordinates.
- **Mapping stays in CarePath:** `serviceCode → service point` (e.g. `LAB → LAB-01`) is CarePath configuration in the `servicepoint` module, not contract data.

Mock-only demo behavior: completing a step readies the next `PENDING` step, and completing the last open step completes the visit (both surface as `visit.updated` events). Seed history replays deterministically on startup.

## CarePath-side ingest (#21)

`apps/api` consumes the feed with a poller (`internal/his/ingest`; interval via `HIS_INGEST_INTERVAL`, default 5s). Each canonical event drives `journey.Service.ApplyHISEvent`, which re-reads the visit snapshot and upserts the CarePath-owned journey projection (`carepath.journey_visit` / `carepath.journey_step`) — the HIS stays the system of record; the projection never guesses state the event payload does not carry. `eventId` is the consumer dedupe key (`carepath.his_applied_event`), so duplicate delivery never duplicates a step, and the feed cursor is kept durably in `carepath.his_ingest_state` so a restart resumes where it left off. Steps whose `serviceCode` has no configured service point are projected with a null `service_point_id` plus a warning log — mapping stays CarePath configuration in the `servicepoint` module.

## Demo-driver API and console (#22)

The console is the demo operator's steering wheel, served by Mock HIS at **`/console`** (one embedded static page — visit list with auto-refresh, Start/Complete buttons per step, create-visit and add-order forms, and a live canonical event tail). It is deliberately **not** part of the canonical contract above: a real HIS has its own operator tooling, so these endpoints exist on the mock only and must never be consumed by CarePath.

- `GET /api/v1/demo/visits` — list all visits (seed + created), for console selection.
- `POST /api/v1/demo/visits` — `{patientRef?, serviceCodes?[]}`. Ids are assigned (`VISIT-NNN`, `PATIENT-DEMO-NNN`); empty codes fall back to the standard 5-step template; the visit opens `ACTIVE` with the first step `READY` and the rest `PENDING`. Emits canonical `visit.opened` + `service.requested` per step.
- `POST /api/v1/demo/visits/{visitId}/orders` — `{serviceCode}` (e.g. `XRAY`): appends a `PENDING` step after the current last sequence; it becomes `READY` when the preceding open step completes. Emits canonical `service.requested`. `409` when the visit is `COMPLETED`/`CANCELLED`.
- `GET /api/v1/demo/visits/{visitId}/qrcode.png` — PNG QR encoding the visit id, shown in the console's visit detail panel (demo prop for getting the id into the patient view by scanning).
- Service completion uses the **canonical transition command** above — the same surface CarePath will use, so step status changes have exactly one path.

Every console action mutates only Mock HIS state and surfaces as canonical events; CarePath learns through the #21 feed poller exclusively — nothing writes the CarePath database directly.

One demo-operational caveat: the event log lives in memory, so restarting Mock HIS resets event ids from `EVT-000001` while CarePath's stored feed cursor keeps its old (higher) value — the poller would then see nothing new. Start a demo from a fresh stack (`docker compose down -v && docker compose up --build`) rather than restarting Mock HIS alone against a warm CarePath database.

## Production replacement

```text
CarePath Core
   -> HIS Port
      -> MockHISAdapter       (local/demo)
      -> RealHISAdapter       (production)
```

No journey-domain module should import a mock-specific type.
