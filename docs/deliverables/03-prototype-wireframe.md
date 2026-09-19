# 3. Prototype / Wireframe (ต้นแบบหน้าจอ)

**Deliverable 3 of 6** per the hackathon brief §10: at least 5 main screens, as a sketch or a design file. This deliverable reuses [`docs/designs/patient-staff-ui.html`](../designs/patient-staff-ui.html) — no rename or relocation needed; `docs/README.md` already references it from `docs/designs/` for engineering purposes, and moving it would only break that link.

## 3.1 What the file is

A self-contained, exported interactive mockup (not a plain static page — it unpacks itself via an embedded manifest on load, so open it through a local server or the file directly in a browser rather than reading its source as HTML). It renders the first 6 boards below: 4 distinct screens, 2 of them shown in both mobile and desktop layout.

Screens 5–7 (§3.2) were added later, directly to the source design canvas the export came from, rather than by re-exporting the bundle — the export embeds every webfont as base64 inside the manifest, and hand-regenerating that is more likely to corrupt the file than to help. They're viewable in the canvas itself (§3.4) and, more usefully, as working React screens in `apps/web` (§3.5).

## 3.2 Screens

| # | Screen | Viewport | Covers |
|---|---|---|---|
| 1 | Patient · Home — journey timeline (registered → screened → **in progress: blood draw**, queue #12, ~8 min wait → next: pharmacy → cashier) | Mobile | [US-02](../requirements/user-stories.md#us-02-see-my-current-and-next-step--must-m4) (M4), [FR-03](../requirements/functional-requirements.md) |
| 2 | Patient · Navigation — route to pharmacy, QR-based "last known location" banner with re-scan action, step chips along the path | Mobile | [US-03](../requirements/user-stories.md#us-03-navigate-to-the-next-service--must-m5-m6), [US-04](../requirements/user-stories.md#us-04-establish-my-current-location--should-s2) (M5/M6, S2), [FR-04](../requirements/functional-requirements.md), [FR-06](../requirements/functional-requirements.md) |
| 3 | Staff · Overview — today's flow: patients in service, top bottleneck, average wait, count with unknown location, service-point wait-time table, "patients needing help" list | Mobile + Desktop | [US-18](../requirements/user-stories.md#us-18-view-bottlenecks-and-average-wait-time--should-s7) (S7), [FR-17](../requirements/functional-requirements.md), [FR-22](../requirements/functional-requirements.md) |
| 4 | Staff · Service-point map — service points by zone/category with live queue count and wait time, plus sub-tabs for building map and patient list | Mobile + Desktop | [US-05](../requirements/user-stories.md#us-05-configure-service-point-mapping--must-m1) (M1), [FR-05](../requirements/functional-requirements.md), [FR-11](../requirements/functional-requirements.md), [FR-17](../requirements/functional-requirements.md) |
| 5 | Login / role selection — username/password (inert, no auth backend yet) plus one of four role cards (registration staff, service-point staff, admin, executive); the primary action signs in to that role's own landing screen | Desktop | [US-17](../requirements/user-stories.md#us-17-manage-user-roles-and-access--must-m8), [FR-18](../requirements/functional-requirements.md) (M8) |
| 6 | Staff · Register patient — look a patient up by HN against (mock) HIS, assign one of three Care Pathway Templates, preview the ordered VisitStep list CarePath will generate before confirming | Desktop | [US-13](../requirements/user-stories.md#us-13-register-a-visit-and-assign-a-pathway-template--must-m3), [FR-13](../requirements/functional-requirements.md), [FR-14](../requirements/functional-requirements.md) (M2/M3) |
| 7 | Staff · Queue-call console — one service point's current ticket (called → arrived → in progress → done), skip, insert-an-unplanned-step, and the shrinking list of who's still waiting | Desktop | [US-14](../requirements/user-stories.md#us-14-call-the-queue-and-record-step-completion--must-m7), [US-15](../requirements/user-stories.md#us-15-insert-an-unplanned-step--must-m7), [FR-15](../requirements/functional-requirements.md), [FR-16](../requirements/functional-requirements.md) (M7) |

Screens 3 and 4 both expose a left-hand section switcher (Overview / Service Points / Building map / Patients) inside the desktop board, so more ground is covered than the 7 top-level boards alone suggest — but the deeper sub-views weren't individually verified pixel-by-pixel for this deliverable.

Screens 5–7 are desktop-only for now (a shared front-desk or counter workstation); a mobile layout for them, in the style already used for screens 3 and 4, is a reasonable next design pass but wasn't required to close the Must-Have gap below.

## 3.3 Gap closed

The three Must-Have flows previously flagged here as missing — login/role selection (FR-18), registration + pathway-template assignment (FR-14), and the service-point queue-call console (FR-15/FR-16) — are now screens 5–7 above. All four Must-Have user-facing flows from [deliverable 1](01-requirement-specification.md) that need a screen at all now have one.

## 3.4 How to view the canvas

Screens 1–4 ship as a self-unpacking export, so open it via a local server rather than double-clicking (some browsers block the unpack script on `file://`):

```bash
cd docs/designs && python3 -m http.server 8731
```

Then open `http://localhost:8731/patient-staff-ui.html`.

Screens 5–7 live only in the source design canvas (a private, interactive link — share it with reviewers who need to open it directly rather than pasting it into a public place). Ask the deliverable's author for the current link if you need to view or edit them there.

## 3.5 Screens 5–7 also exist as working code

Unlike screens 1–4, which are wireframe-only, screens 5–7 were built straight through into real React routes in `apps/web` — `/login`, `/staff/register`, `/staff/queue` — using the CarePath design system (`apps/web/src/design-system/`, spec in [`DESIGN.md`](../../DESIGN.md)) rather than the canvas's own markup. They render on `/design` alongside the rest of the component catalogue. None of the three call a real backend yet (no auth, registration, or queue-call endpoint exists in `packages/contracts/openapi/carepath.yaml`) — each screen runs on local component state and the same `apps/web/src/mocks/demo-data.ts` the staff console already uses, exactly like the deliverable-4 gap that remains for those endpoints.
