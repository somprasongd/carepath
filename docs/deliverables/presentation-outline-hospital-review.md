# Presentation Outline — Hospital/Stakeholder Review

Direction from Ball-SD (2026-09-20) for this round of presentation, distinct from the hackathon judge pitch in [06. Presentation Slides](06-presentation-slides.md): lead with the problem, show how the solution is designed to work *with* the hospital's existing HIS, keep the HIS as owner of visit/order data, walk the patient flow in depth, only list what staff gets, add an executive-dashboard section with rationale, cover the referral-slip QR idea, demo only through Mock HIS with a fresh VN, and present the technical blueprint. Content is pulled from existing docs/ADRs and cross-checked against the current code (verified 2026-09-20 — see file/line pointers inline where a claim is code-verified, not just doc-derived).

Slide deck built from this outline: [presentation-outline-hospital-review.pptx](presentation-outline-hospital-review.pptx) (14 slides, 7:00 timing, speaker notes per slide).

## Timing budget (7:00 total)

Same hard stop as the hackathon pitch. The extra required content (HIS-integration depth, executive dashboard, referral-slip QR) is new relative to [06. Presentation Slides](06-presentation-slides.md), so the live-demo window is intentionally shorter here (2:30 vs. 2:50) to make room.

| # | Section | Time | Cumulative |
|---|---|---|---|
| 1 | Title | 0:10 | 0:10 |
| 2 | Problem (§1.1) | 0:30 | 0:40 |
| 3 | Solution & HIS integration design (§1.2–1.5) | 1:00 | 1:40 |
| 4 | Patient flow walkthrough, narrated (§2.1) | 0:30 | 2:10 |
| 5 | Staff (list only) + executive dashboard & why (§2.2–2.3) | 0:40 | 2:50 |
| 6 | ใบนำทาง QR idea + relative sharing — what & why (§3) | 0:40 | 3:30 |
| — | **Live demo** — Mock-HIS-driven, fresh VN (§4) | 2:30 | 6:00 |
| 7 | Technical blueprint (§5) | 0:45 | 6:45 |
| 8 | Closing / thank you | 0:15 | 7:00 |

Rehearsal notes:
- Section 3 (HIS integration) and the live demo are the two sections most likely to run long — they carry the most new ground for this audience. If time slips, cut from the technical blueprint (§5) first, not from HIS integration or the demo; the blueprint is the one section with a standalone doc the audience can read afterward.
- Section 4 (§2.1 patient-flow walkthrough) can be trimmed or folded into the live demo's narration if rehearsal shows the two are repeating each other — the demo already shows the same flow live.

## 1. Problem → Solution → HIS Integration Design

### 1.1 Problem (from the brief)

- ผู้ป่วยหลงทางข้ามอาคาร/ชั้น โดยเฉพาะผู้สูงอายุ ผู้ใช้รถเข็น และผู้มาครั้งแรก
- ไม่รู้ว่าเหลืออีกกี่ขั้นตอน ต้องรออีกนานแค่ไหน ขั้นตอนถัดไปอยู่ที่ใด
- เจ้าหน้าที่เสียเวลาตอบคำถามเรื่องเส้นทางแทนที่จะดูแลผู้ป่วย

Source: primary hospital/patient goals in [Product Requirements](../requirements/product-requirements.md) — "reduce navigation confusion," "reduce repetitive wayfinding questions to staff," "make patient flow visible **without replacing the HIS**."

### 1.2 Solution shape

CarePath = ผู้ช่วยนำทางส่วนตัวตลอดการมาโรงพยาบาลหนึ่งครั้ง โดยแยกสองเรื่องออกจากกันโดยเจตนา: **Care Graph** (ขั้นตอนถัดไปคืออะไร) กับ **Navigation Graph** (จะไปถึงอย่างไร) เชื่อมกันผ่าน `ServicePoint → Place` ([ADR-0002](../adr/0002-separate-care-and-navigation-graphs.md), [System Architecture](../architecture/system-architecture.md)).

### 1.3 HIS owns visit/order data — say this explicitly

Ownership table, straight from [ADR-0009](../adr/0009-carepath-owns-journey-plan.md):

| Concern | Owner |
|---|---|
| Visit, clinic assignment, orders, order results, encounter completion | **HIS** (system of record) |
| Journey plan: which steps exist, their order, their status | **CarePath** (derived) |
| Service point ↔ place binding, navigation | **CarePath** |

CarePath never writes HIS-owned status. It only derives a journey from facts the HIS already produces.

### 1.4 How the connection is designed

- **No direct DB access.** CarePath integrates only through an internal HIS adapter/port; it must never query the HIS database directly ([ADR-0005](../adr/0005-his-adapter-and-mock-his.md)). Mock HIS implements the same conceptual contract today, so swapping in a real HIS later touches only the adapter — journey and navigation code do not change.
- **Canonical facts, not vendor payloads.** The HIS emits 8 canonical facts CarePath consumes — `visit.opened`, `visit.updated`, `visit.closed`, `order.placed`, `order.performed`, `order.resulted`, `order.cancelled`, `encounter.completed` — over a REST event feed, idempotent on `eventId` ([ADR-0008](../adr/0008-his-canonical-event-contract.md) as amended by [ADR-0009](../adr/0009-carepath-owns-journey-plan.md)). A real HIS adapter maps whatever it has (webhook, polling, message queue) onto these same 8 facts without touching domain code.
- **Why facts, not steps:** the HIS has no concept of an ordered patient journey — it doesn't know what "next" means. Asking a real hospital's HIS team to model that would be a much bigger integration ask than asking it to emit visit/order/encounter facts it already has (ADR-0009 §Context).

### 1.5 How CarePath is designed (the part that turns facts into a journey)

- The journey plan is a **pure function** `plan(visitFacts) -> []Step`, deterministic and side-effect free, recomputed and diffed against the stored plan on every inbound fact — not appended to (ADR-0009 §3).
- It reproduces the real OPD flow the hospital walked us through: registration always first; pre-visit orders (lab/X-ray) before the clinic for appointment patients; a mid-consultation order sends the patient out and back to the **same doctor**; cashier exactly once regardless of clinic count; pharmacy only if a drug was ordered.
- Backend shape enforcing this stays a modular monolith with hexagonal modules — `journey`, `his` (integration adapter + event-feed poller), `servicepoint`, `navigation`, `location` — each with inward-pointing dependencies and its own Postgres adapter ([ADR-0001](../adr/0001-monorepo-modular-monolith.md), [ADR-0007](../adr/0007-hexagonal-modules-transaction-in-context.md), full module list in [Technical Blueprint](../architecture/technical-blueprint.md#backend-modules)).

## 2. Patient Flow (deep dive), Staff (list only), Executive Dashboard (with why)

### 2.1 Patient flow — walk through in detail

Use the [System Architecture](../architecture/system-architecture.md) diagram plus the live demo (section 4) to show, in order:

1. Patient opens CarePath (LINE LIFF or VN lookup) → sees the journey timeline: completed / current / upcoming steps (FR-01–FR-03).
2. Every actionable step resolves to one recommended destination — a `ServicePoint` → `Place` ([ADR-0002](../adr/0002-separate-care-and-navigation-graphs.md)).
3. No GPS indoors — current position comes from QR (baseline) or Zigbee (optional phase 2), same provider interface either way ([ADR-0004](../adr/0004-location-provider-abstraction.md)); route renders on the SVG floor plan once position is known.
4. The timeline updates itself (polling today) as HIS facts and staff actions change the plan — no manual refresh, no re-asking staff "what's next."

### 2.2 Staff — list only, don't deep-dive

State what exists, briefly, no walkthrough:

- Staff/admin login with role-based access — argon2id passwords, JWT access + rotating refresh token (`POST /auth/login`, `apps/api/internal/auth`, [ADR-0010](../adr/0010-staff-auth-jwt-argon2.md)).
- Visit list for registration/service-point staff (`GET /staff/visits`) — browse open visits; this is a list, not a search-by-VN lookup, and it's not a create-visit screen — the HIS still opens the visit.
- Service-point console: start/complete a step (`POST /journeys/:visitId/steps/:stepKey/transition`), override the return-to-doctor inference when it's wrong (`POST /journeys/:visitId/clinics/:clinicCode/close-round`, ADR-0009 §4).
- (Demo-only, not a hospital-staff surface) Mock HIS console — stands in for the real HIS during development; a real deployment would not include it.

Note: FR-16 ("unplanned step insertion," Must/M7) is in the requirements doc but not implemented in code as of this check — `TransitionStep` only mutates a step already in the planner's output, there is no add-a-step endpoint. Don't claim this one live if asked "can staff add a step?"

### 2.3 Executive dashboard — include it, and say why

- What it shows: average wait time per service point, bottleneck service points, average total time a patient spends in the hospital (FR-22 / US-18, `Should — S7`).
- Built on the `analytics` module (#86) — `GET /api/v1/analytics/overview`, EXECUTIVE-role-only (`apps/api/internal/analytics/handler.go`) — read-only aggregates over the append-only `journey_step_status_event` timeline that the journey module already writes for its own diffing; the dashboard adds no new data collection. Web screen: `apps/web/src/features/analytics` + the staff overview route.
- **Why present this to hospital leadership specifically:** it converts an operational side-effect of the patient-flow feature into decision support — where to add staff, which service point is the actual bottleneck — without asking the hospital to instrument anything new. It's the argument for why an executive sponsor should care about a "patient navigation app" beyond patient experience.

## 3. Referral Slip (ใบนำทาง) QR + Sharing With Relatives

Be explicit about what's built today vs. proposed, per the brief's honesty expectation (see [06. Presentation Slides §8](06-presentation-slides.md)).

### 3.1 The idea (proposed, not yet built)

The hospital's existing printed **ใบนำทาง** (referral/queue slip handed to the patient at registration) could carry a QR code that deep-links straight into that patient's CarePath journey (VN lookup, FR-02) — replacing the manual "ค้นหาการนัดหมาย" search step in the demo today with a scan. This is a natural extension of the existing QR location-provider pattern (FR-06, [ADR-0004](../adr/0004-location-provider-abstraction.md)); it's a new entry-point QR rather than a new subsystem.

### 3.2 What can be shared with relatives, and why (this part is built — ADR-0011)

The patient can generate a **share link** from their own journey screen. What a relative sees, opening it with no login:

- current step as human-readable Thai text (e.g. "รอเจาะเลือด"),
- a coarse status only — `WAITING` / `IN_SERVICE` / `DONE`,
- the service point's display name and floor,
- when it was last updated, and when the link expires.

What is deliberately **not** in the shared view, by construction (data minimisation, [ADR-0011](../adr/0011-visit-share-link.md)): patient name, HN/VN, `visitId`, the internal step key (it embeds the clinic code), clinic code, order references, and the full step list.

**Why this shape:**

- A relative isn't a person CarePath knows — no login, so the link itself has to carry just enough access and nothing else.
- It must survive being forwarded through LINE/chat apps, which are outside our control the moment the patient taps "share" — so it's read-only, expires by default in 4 hours, and the patient can revoke it ("หยุดแชร์") at any time.
- The token never appears in a URL path or query string — it rides in the URL **fragment**, which browsers never send to a server, so it can't leak into request logs (NFR-08).

## 4. Demo — Mock HIS Driven, Fresh VN

Ball-SD's ask: drive the demo mainly through Mock HIS — open a new VN as an appointment with a pre-visit order, place an additional order after the doctor visit, end at cashier + pharmacy, and show the map at the relevant points. This is a leaner shape than the full choreography in [07. Demo Script](07-demo-script.md); reuse that script's verified commands/expected results, but sequence them as follows and open a **new** VN live instead of reusing seeded `VISIT-002`, so the audience sees the plan being derived, not replayed:

1. **Mock HIS console** — open a new visit as an **appointment**, assign clinic (e.g. อายุรกรรม/MED), with a pre-visit order already placed (e.g. X-ray) → this exercises ADR-0009's rule "appointment patients with pre-visit orders go to LAB/X-ray first, then the clinic."
2. **Patient app** — show the derived journey timeline: the pre-visit order step appears before the clinic step, automatically, with no step manually configured.
3. **Map** — resolve the patient's position (QR or Zigbee simulator, per [07. Demo Script Beat 3](07-demo-script.md#beat-3--รายงานตำแหน่งปัจจุบัน-030)) and show the route to the first destination.
4. **Mock HIS console** — mark the pre-visit order performed + resulted; patient app flips to the clinic step becoming actionable; show the route updating to the clinic.
5. **Mock HIS console** — mid-encounter, place an **additional** order (e.g. a lab test ordered by the doctor) → this exercises "mid-consultation the doctor may order more investigations — the patient does them and comes back to the same doctor" (ADR-0009 §Context, rule 4). Show the timeline growing a `WAITING` step and a follow-up clinic step appearing.
6. **Mock HIS console** — complete the encounter → journey resolves to cashier (always exactly once, regardless of clinic count) then pharmacy (only because a drug order exists, if one was added) → show the map/route for each.
7. Close on the same honesty note as the existing demo: this whole path is pinned by the API-level E2E test in CI ([`apps/api/internal/e2e`](../../apps/api/internal/e2e)), so what's demoed is what's tested every day.

Reuse [07. Demo Script](07-demo-script.md)'s pre-demo checklist (reset seed, confirm health, open tabs, backup recording) — a fresh-VN demo still needs the same environment prep, it just doesn't rely on the pre-seeded `VISIT-002` state.

## 5. Technical Blueprint

Present [Technical Blueprint](../architecture/technical-blueprint.md) directly — it's the canonical description, and its module list/runtime-chain/location-abstraction sections were code-verified. Its frontend stack list was **not** accurate as of this check and has been corrected in that doc (React Hook Form, Zod, Zustand, and TanStack Table are listed there as baseline but are not in `apps/web/package.json` — no matching dependency). Suggested cut for a stakeholder audience (skip deep module internals unless asked):

- **Stack at a glance**: React 19 + TypeScript + Vite frontend (Tailwind CSS + Radix/shadcn-style components, TanStack Query + Router), LINE LIFF for the patient channel; Go/Fiber v3 backend, PostgreSQL, OpenAPI-contract-first; Docker Compose today, Kubernetes-ready boundaries for later.
- **Core runtime chain** (one diagram, the clearest "how it fits together" visual):
  ```
  Visit facts (clinics, orders, encounters)
    -> Journey plan (ADR-0009)
    -> ServicePoint
    -> Place
    -> NavigationNode
    -> Route
  ```
- **Modular monolith, not microservices** — intentional for coordination speed at this stage, with hexagonal boundaries per module so pulling one out later (e.g. a real HIS adapter, or a separate analytics service) doesn't require a rewrite ([ADR-0001](../adr/0001-monorepo-modular-monolith.md), [ADR-0007](../adr/0007-hexagonal-modules-transaction-in-context.md)).
- **Location abstraction** — `LocationService` behind one interface with QR / Zigbee / Manual providers; this is the same pattern that lets the referral-slip QR idea (section 3.1) slot in without new plumbing.

## Open items before this goes to slides

- Section 3.1 (referral-slip QR) is a proposal, not shipped — confirm whether to present it as a roadmap item or scope it down if asked "is this built?"
- Confirm which staff-console screens to show live vs. only list, per section 2.2 — note the corrected list above (visit list, not VN lookup; no unplanned-step-insertion feature exists yet).
- `docs/architecture/technical-blueprint.md`'s frontend dependency list was stale (claimed React Hook Form/Zod/Zustand/TanStack Table that aren't in `package.json`) and has been corrected as part of this check — re-verify before the next presentation round if the frontend stack changes again.
