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
| 5 | Login — username/password. Drawn with four role cards (registration staff, service-point staff, admin, executive) and inert fields; **[ADR-0010](../adr/0010-staff-auth-jwt-argon2.md) supersedes that shape** — the role picker is removed (a user does not choose their own privileges), the form posts real credentials, and the landing screen is derived from the role in the returned token | Desktop | [US-22](../requirements/user-stories.md#us-22-log-in-to-the-staff-console--must-m8--added-by-adr-0010), [US-17](../requirements/user-stories.md#us-17-manage-user-roles-and-access--must-m8--mvp-slice-scoped-by-adr-0010), [FR-18](../requirements/functional-requirements.md) (M8) |
| 6 | Staff · Pathway-template checklist — pick one of three Care Pathway Templates and read off its ordered service-code checklist, with a copy-to-clipboard action, for entering into the HIS when opening the patient's visit there | Desktop | [US-13](../requirements/user-stories.md#us-13-look-up-a-patients-derived-visit-plan--must-m3--revised-by-adr-0009), [FR-13](../requirements/functional-requirements.md), [FR-14](../requirements/functional-requirements.md) (M2/M3) |
| 7 | Staff · Queue-call console — one service point's current ticket (called → arrived → in progress → done), skip, insert-an-unplanned-step, and the shrinking list of who's still waiting | Desktop | [US-14](../requirements/user-stories.md#us-14-call-the-queue-and-record-step-completion--must-m7), [US-15](../requirements/user-stories.md#us-15-confirm-whether-a-patient-returns-after-an-extra-test--must-m7--revised-by-adr-0009), [FR-15](../requirements/functional-requirements.md), [FR-16](../requirements/functional-requirements.md) (M7) |

Screens 3 and 4 both expose a left-hand section switcher (Overview / Service Points / Building map / Patients) inside the desktop board, so more ground is covered than the 7 top-level boards alone suggest — but the deeper sub-views weren't individually verified pixel-by-pixel for this deliverable.

Screens 5–7 have both a mobile (390×844) and a desktop (1440×900) reference frame in `/design`, built as one component per screen that adapts with a container query rather than two separate designs — the table above cites the desktop viewport since that's where a shared front-desk or counter workstation actually gets used.

## 3.3 Gap closed

The three Must-Have flows previously flagged here as missing — login/role selection (FR-18), pathway-template lookup (FR-13/FR-14), and the service-point queue-call console (FR-15/FR-16) — are now screens 5–7 above. All four Must-Have user-facing flows from [deliverable 1](01-requirement-specification.md) that need a screen at all now have one.

**Screen 6 changed shape after ADR-0008.** It originally let staff look a patient up by HN and "register" their visit, with CarePath generating the `VisitStep` list. [ADR-0008](../adr/0008-his-canonical-event-contract.md) (accepted the same day) settled that the HIS is the sole system of record for opening a visit and ordering services — CarePath only projects derived state from the events HIS reports. Screen 6 was reworked to fit: it no longer looks up a patient or "registers" anything; it lets staff pick a Pathway Template and read off (or copy) the ordered service-code checklist to enter into the HIS themselves. FR-13, FR-14, and US-13 were reworded to match (see `docs/requirements/`); the canvas mockup was not re-exported to match, so treat the working screen (§3.5) as authoritative for screen 6.

**Screen 5 changed shape after [ADR-0010](../adr/0010-staff-auth-jwt-argon2.md).** It was drawn while no auth backend existed, so it let the user pick their own role from four cards — a demo affordance, not a design intent. ADR-0010 makes login real (argon2id password, JWT access + rotating refresh token) and roles authoritative: the console reads `STAFF` or `ADMIN` out of the token and lands the user accordingly, so the cards come out and the form gains a pending state and an inline "ชื่อผู้ใช้หรือรหัสผ่านไม่ถูกต้อง" error that deliberately does not say which half was wrong. The demo accounts are `admin`/`demo` and `staff`/`demo`.

**Screen 6's premise changed again under [ADR-0009](../adr/0009-carepath-owns-journey-plan.md).** ADR-0008 assumed the HIS could still report *an ordered step list* even though CarePath must not write it; ADR-0009 found the HIS has no such list to report at all — it only has visits, clinic assignments, and orders. "Pick a Pathway Template and copy its service-code checklist" no longer matches anything the HIS can consume, since there is no HIS field to paste those codes into. The revised FR-14/US-13 shape is "look up a visit by VN and see the plan CarePath's journey planner derived for it." The working screen (`apps/web/src/routes/staff/-PathwayTemplates.tsx`) still shows the pre-ADR-0009 checklist UI — it reads only static local config and calls no backend, so nothing is functionally broken by this, but it is a known, tracked follow-up rework, not a screen to treat as representative of the current design intent.

## 3.4 How to view the canvas

Screens 1–4 ship as a self-unpacking export, so open it via a local server rather than double-clicking (some browsers block the unpack script on `file://`):

```bash
cd docs/designs && python3 -m http.server 8731
```

Then open `http://localhost:8731/patient-staff-ui.html`.

Screens 5–7 live only in the source design canvas (a private, interactive link — share it with reviewers who need to open it directly rather than pasting it into a public place). Ask the deliverable's author for the current link if you need to view or edit them there.

## 3.5 Screens 5–7 also exist as working code

Unlike screens 1–4, which are wireframe-only, screens 5–7 were built straight through into real React routes in `apps/web` — `/login`, `/staff/pathway-templates`, `/staff/queue` — using the CarePath design system (`apps/web/src/design-system/`, spec in [`DESIGN.md`](../../DESIGN.md)) rather than the canvas's own markup. They render on `/design` alongside the rest of the component catalogue. None of the three call a real backend yet — each screen runs on local component state and the same `apps/web/src/mocks/demo-data.ts` the staff console already uses. That is now a gap rather than an absence for `/login`: `POST /api/v1/auth/login|refresh|logout` and `GET /api/v1/auth/me` are specified in `packages/contracts/openapi/carepath.yaml` and designed in ADR-0010, and wiring the screen to them is the next slice. No queue-call endpoint exists at all. `/staff/pathway-templates` never will call one for its own purposes: per ADR-0008 it only reads static Pathway Template config, it never writes to HIS.
