# ADR-0008: Canonical HIS Event/Command Contract

- Status: Accepted
- Date: 2026-09-19

## Context

Issue #20 requires a canonical HIS↔CarePath integration contract before the HIS adapter boundary (#21) and the service-step transition API (#19) are built. Today the only integration surface is the pull snapshot `GET /api/v1/visits/{visitId}` (`packages/contracts/openapi/mock-his.yaml` v0.1) consumed through `apps/api/internal/his.Client`. Two questions had to be settled first: how state changes flow between HIS and CarePath, and who owns visit/step status — the SDLC workflow flagged visit-state ownership as an ADR-worthy decision that must exist before the transition API is built.

## Decision

1. The integration contract is a **canonical event/command model**, not a transport and not a vendor payload:
   - Events (HIS → CarePath): `visit.opened`, `visit.updated`, `service.requested`, `service.started`, `service.completed`, `service.cancelled` — one envelope (`eventId`, `occurredAt`, `visitId`, `patientRef`, `type`, typed payload).
   - Commands (CarePath → HIS): service-step status transitions (`STARTED`, `COMPLETED`, `CANCELLED`) carrying a caller-assigned `commandId`.
2. **The HIS is the system of record for visit and service-step status.** CarePath never writes HIS-owned status directly: staff actions taken in CarePath are forwarded as commands through the HIS port, and their effect is re-read as snapshots/events. CarePath owns only derived state — the journey projection, current/next-step resolution, service-point binding, and navigation.
3. **Idempotency:** `eventId` (HIS-assigned, unique) identifies events for consumers; `commandId` (caller-assigned) lets the HIS deduplicate command retries; a transition to the step's current state is a no-op success.
4. **MVP transport is REST per ADR-0006:** the existing pull snapshot stays, plus a step-transition command endpoint and an append-only event feed on Mock HIS. A real HIS adapter may map webhooks, polling, or messages onto the same canonical model (per ADR-0005) without touching journey-domain code.
5. External identifiers (`visitId`, `patientRef`, optional `orderRef`) are HIS-assigned and opaque; `patientRef` carries no PHI beyond the opaque reference.
6. Canonical enums are defined once in the contract: visit status `ACTIVE | COMPLETED | CANCELLED`; step status `PENDING | READY | STARTED | COMPLETED | CANCELLED`. Vendor-specific codes are translated to these enums inside the HIS adapter.
7. Mapping an external service code to a CarePath service point (`LAB → LAB-01`) is CarePath configuration (the `servicepoint` module); contracts carry only the external `serviceCode`.

## Consequences

- Journey-domain code depends only on the canonical model, never on a vendor HIS payload or mock-specific shapes.
- #19's transition API becomes a proxy over the HIS port command; CarePath keeps no local step-status writes.
- Mock HIS gains a transition endpoint and an event log (also the backend for the #22 console, and replayable for demos).
- The CarePath-side domain statuses of #17 must map to the canonical enums (e.g. `in_progress` ← `STARTED`).
- Adding push/webhook later touches only the adapter and Mock HIS, not the canonical contract.
