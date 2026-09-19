# 3. Prototype / Wireframe (ต้นแบบหน้าจอ)

**Deliverable 3 of 6** per the hackathon brief §10: at least 5 main screens, as a sketch or a design file. This deliverable reuses [`docs/designs/patient-staff-ui.html`](../designs/patient-staff-ui.html) — no rename or relocation needed; `docs/README.md` already references it from `docs/designs/` for engineering purposes, and moving it would only break that link.

## 3.1 What the file is

A self-contained, exported interactive mockup (not a plain static page — it unpacks itself via an embedded manifest on load, so open it through a local server or the file directly in a browser rather than reading its source as HTML). It renders 6 boards: 4 distinct screens, 2 of them shown in both mobile and desktop layout.

## 3.2 Screens

| # | Screen | Viewport | Covers |
|---|---|---|---|
| 1 | Patient · Home — journey timeline (registered → screened → **in progress: blood draw**, queue #12, ~8 min wait → next: pharmacy → cashier) | Mobile | [US-02](../requirements/user-stories.md#us-02-see-my-current-and-next-step--must-m4) (M4), [FR-03](../requirements/functional-requirements.md) |
| 2 | Patient · Navigation — route to pharmacy, QR-based "last known location" banner with re-scan action, step chips along the path | Mobile | [US-03](../requirements/user-stories.md#us-03-navigate-to-the-next-service--must-m5-m6), [US-04](../requirements/user-stories.md#us-04-establish-my-current-location--should-s2) (M5/M6, S2), [FR-04](../requirements/functional-requirements.md), [FR-06](../requirements/functional-requirements.md) |
| 3 | Staff · Overview — today's flow: patients in service, top bottleneck, average wait, count with unknown location, service-point wait-time table, "patients needing help" list | Mobile + Desktop | [US-18](../requirements/user-stories.md#us-18-view-bottlenecks-and-average-wait-time--should-s7) (S7), [FR-17](../requirements/functional-requirements.md), [FR-22](../requirements/functional-requirements.md) |
| 4 | Staff · Service-point map — service points by zone/category with live queue count and wait time, plus sub-tabs for building map and patient list | Mobile + Desktop | [US-05](../requirements/user-stories.md#us-05-configure-service-point-mapping--must-m1) (M1), [FR-05](../requirements/functional-requirements.md), [FR-11](../requirements/functional-requirements.md), [FR-17](../requirements/functional-requirements.md) |

Screens 3 and 4 both expose a left-hand section switcher (Overview / Service Points / Building map / Patients) inside the desktop board, so more ground is covered than the 6 top-level boards alone suggest — but the deeper sub-views weren't individually verified pixel-by-pixel for this deliverable.

## 3.3 Known gap — screens not yet designed

Three Must-Have flows from [deliverable 1](01-requirement-specification.md) have no screen yet. Flagging explicitly rather than silently shipping fewer than the full Must-Have set:

| Missing screen | Requirement | Priority |
|---|---|---|
| Login / role selection | [FR-18](../requirements/functional-requirements.md) (M8) | Must |
| Registration + pathway-template assignment | [FR-14](../requirements/functional-requirements.md) (M3) | Must |
| Service-point staff queue-call / mark-step-done console | [FR-15](../requirements/functional-requirements.md), [FR-16](../requirements/functional-requirements.md) (M7) | Must |

These should be designed and added before the working-software demo (deliverable 4) if time allows — they're on the critical path for the Must-Have acceptance bar the brief scores against (§11, criterion 1 and 3), not just documentation completeness.

## 3.4 How to view it

The file is a self-unpacking export, so open it via a local server rather than double-clicking (some browsers block the unpack script on `file://`):

```bash
cd docs/designs && python3 -m http.server 8731
```

Then open `http://localhost:8731/patient-staff-ui.html`.
