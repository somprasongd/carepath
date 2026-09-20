# Mock HIS API

Base path: `/api/v1`

Canonical contract: [`packages/contracts/openapi/mock-his.yaml`](../../packages/contracts/openapi/mock-his.yaml) (v0.3.0) — if this page and the contract disagree, the contract wins. Design rationale: [ADR-0008](../adr/0008-his-canonical-event-contract.md), amended by [ADR-0009](../adr/0009-carepath-owns-journey-plan.md) — the HIS has no concept of an ordered journey; it reports visit/order/encounter facts and CarePath derives the plan.

## Health

`GET /health`

## Get visit (snapshot)

`GET /api/v1/visits/{visitId}`

Example:

```json
{
  "visitId": "VISIT-001",
  "patientRef": "PATIENT-DEMO-001",
  "patientName": "สมชาย ใจดี",
  "visitType": "APPOINTMENT",
  "status": "ACTIVE",
  "clinics": [{"code": "MED", "name": "อายุรกรรม"}],
  "orders": [
    {
      "orderRef": "ORD-001",
      "orderType": "LAB",
      "orderName": "CBC",
      "orderedByClinic": "MED",
      "orderedAt": "2026-09-19T08:50:00+07:00",
      "status": "PLACED",
      "performedAt": null,
      "resultedAt": null
    }
  ],
  "openedAt": "2026-09-19T09:00:00+07:00"
}
```

Unknown visits return HTTP 404.

## Event feed

`GET /api/v1/events?after={eventId}&limit={n}`

Append-only feed of canonical HIS facts, oldest first. `after` is an
exclusive cursor: pass the last `eventId` received; `nextAfter` in the
response is the cursor for the next page (empty when done). Default limit 50,
max 200. This is what CarePath's ingest poller consumes to drive the journey
planner.

```json
{
  "events": [
    {
      "eventId": "EVT-000001",
      "occurredAt": "2026-09-19T09:00:00+07:00",
      "visitId": "VISIT-001",
      "patientRef": "PATIENT-DEMO-001",
      "type": "visit.opened",
      "payload": {"patientName": "สมชาย ใจดี", "visitType": "APPOINTMENT", "clinics": [{"code": "MED"}]}
    }
  ],
  "nextAfter": "EVT-000001"
}
```

Event types: `visit.opened`, `visit.updated`, `visit.closed`, `order.placed`,
`order.performed`, `order.resulted`, `order.cancelled`, `encounter.completed`.

## Demo driver (operator console)

Mock HIS has no real front-end of its own — these endpoints stand in for the
HIS screens where the facts above originate (registration, order entry, the
lab confirming a result). They power `GET /console` and are what e2e tests
drive to build a scenario from scratch.

| Endpoint | Emits | Notes |
| --- | --- | --- |
| `GET /api/v1/demo/visits` | — | List every visit |
| `POST /api/v1/demo/visits` | `visit.opened` | Open a visit: `{visitType, clinics: [{clinicCode}], patientRef?, patientName?, orders?}` |
| `POST /api/v1/demo/visits/{visitId}/clinics` | `visit.updated` | Assign an additional clinic mid-visit: `{clinicCode, clinicName?}` |
| `POST /api/v1/demo/visits/{visitId}/orders` | `order.placed` | Place an order: `{orderType, orderName, orderedByClinic}` |
| `POST /api/v1/demo/orders/{orderRef}/performed` | `order.performed` | Procedure done (blood drawn, film taken); requires PLACED |
| `POST /api/v1/demo/orders/{orderRef}/resulted` | `order.resulted` | Result reported; requires PERFORMED |
| `POST /api/v1/demo/orders/{orderRef}/cancel` | `order.cancelled` | Requires not yet RESULTED |
| `POST /api/v1/demo/visits/{visitId}/clinics/{clinicCode}/complete-encounter` | `encounter.completed` | This doctor is done with this patient for this round |
| `POST /api/v1/demo/visits/{visitId}/complete` | `visit.closed` (status COMPLETED) | Requires ACTIVE |
| `POST /api/v1/demo/visits/{visitId}/cancel` | `visit.closed` (status CANCELLED) + `order.cancelled` per open order | No-op if already CANCELLED; rejected if COMPLETED |
| `GET /api/v1/demo/visits/{visitId}/qrcode.png` | — | QR code encoding the patient-view journey link (`<patientAppBaseURL>/patient/journey?visit=<id>`), for the patient hand-off demo prop |

`orderType` is one of `LAB`, `XRAY`, `EKG`, `US`, `DRUG`. A `DRUG` order has no
PERFORMED/RESULTED distinction — CarePath treats it as done once PLACED.
