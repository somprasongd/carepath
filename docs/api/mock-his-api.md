# Mock HIS API

Base path: `/api/v1`

Canonical contract: [`packages/contracts/openapi/mock-his.yaml`](../../packages/contracts/openapi/mock-his.yaml) (v0.2.0) — if this page and the contract disagree, the contract wins. Design rationale: [ADR-0008](../adr/0008-his-canonical-event-contract.md).

## Health

`GET /health`

## Get visit (snapshot)

`GET /api/v1/visits/{visitId}`

Example:

```json
{
  "visitId": "VISIT-001",
  "patientRef": "PATIENT-DEMO-001",
  "status": "ACTIVE",
  "steps": [
    {"sequence": 1, "serviceCode": "REGISTRATION", "status": "COMPLETED"},
    {"sequence": 2, "serviceCode": "SCREENING", "status": "COMPLETED"},
    {"sequence": 3, "serviceCode": "DOCTOR", "status": "COMPLETED"},
    {"sequence": 4, "serviceCode": "LAB", "status": "READY"},
    {"sequence": 5, "serviceCode": "PHARMACY", "status": "PENDING"}
  ]
}
```

Unknown visits return HTTP 404.

## Transition a service step (command)

`POST /api/v1/visits/{visitId}/steps/{sequence}/transition`

The only way step status changes. The HIS is the system of record (ADR-0008):
CarePath sends this command when staff act in its UI; it never writes step
status itself. `commandId` is the caller-assigned idempotency key — replaying
an applied command, or transitioning to the step's current status, is a no-op
success.

```json
{"commandId": "44444444-4444-4444-4444-444444444444", "to": "STARTED"}
```

`to` is one of `STARTED`, `COMPLETED`, `CANCELLED`. The response is the
updated step:

```json
{"sequence": 4, "serviceCode": "LAB", "status": "STARTED"}
```

Errors: 400 invalid body or unknown target; 404 unknown visit/step; 409
illegal transition (e.g. from a terminal `COMPLETED`/`CANCELLED` status).

Mock-only behavior worth knowing for demos: completing a step also readies
the next `PENDING` step, and completing the last open step completes the
visit — both emitted as `visit.updated` events.

## Event feed

`GET /api/v1/events?after={eventId}&limit={n}`

Append-only feed of canonical HIS events, oldest first. `after` is an
exclusive cursor: pass the last `eventId` received; `nextAfter` in the
response is the cursor for the next page (empty when done). Default limit 50,
max 200.

```json
{
  "events": [
    {
      "eventId": "EVT-000001",
      "occurredAt": "2026-09-19T09:01:00+07:00",
      "visitId": "VISIT-001",
      "patientRef": "PATIENT-DEMO-001",
      "type": "visit.opened",
      "payload": {"status": "ACTIVE"}
    }
  ],
  "nextAfter": "EVT-000001"
}
```

Event types: `visit.opened`, `visit.updated`, `service.requested`,
`service.started`, `service.completed`, `service.cancelled`. The seed visit
replays its history on startup (1 opened + 5 requested + 3 completed = 9
events) with deterministic timestamps; transitions append runtime events.
