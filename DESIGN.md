---
version: alpha
name: CarePath
description: >
  Hospital patient-journey and indoor navigation UI. Two audiences share one
  visual language: patients (mobile, LINE LIFF, Thai-first, one-thumb use)
  and staff (mobile-first, scales to a desktop console). The design system is
  deliberately borrowed from the product's own SVG floor plans
  (packages/floorplans) rather than invented separately, so the app chrome
  and the hospital wayfinding signage it renders are visibly the same system.
colors:
  primary: "#ff7a00"
  primary-tint: "#fff1e2"
  secondary: "#0b6e99"
  secondary-tint: "#e3f1f7"
  ink: "#1f2b33"
  ink-muted: "#66727d"
  neutral: "#f4f6f5"
  surface: "#ffffff"
  line: "#dde3e3"
  success: "#1f7a5c"
  success-tint: "#e7f5ef"
  warning: "#9a5310"
  warning-tint: "#fff1e2"
  zone-public: "#fff7df"
  zone-opd: "#dcecff"
  zone-diagnostic: "#e7e0ff"
  zone-pharmacy: "#e4f7df"
  zone-rehab: "#fff1c9"
  zone-ipd: "#dff5f4"
  zone-support: "#eceff2"
typography:
  display-stat:
    fontFamily: Space Grotesk
    fontSize: 34px
    fontWeight: 700
    lineHeight: 1.1
  h1:
    fontFamily: Noto Sans Thai
    fontSize: 26px
    fontWeight: 700
    lineHeight: 1.28
  h2:
    fontFamily: Noto Sans Thai
    fontSize: 16px
    fontWeight: 700
    lineHeight: 1.3
  body-md:
    fontFamily: Noto Sans Thai
    fontSize: 14px
    fontWeight: 400
    lineHeight: 1.6
  body-sm:
    fontFamily: Noto Sans Thai
    fontSize: 13px
    fontWeight: 400
    lineHeight: 1.5
  label-code:
    fontFamily: Space Grotesk
    fontSize: 13px
    fontWeight: 700
    lineHeight: 1
  caption:
    fontFamily: Noto Sans Thai
    fontSize: 12px
    fontWeight: 500
    lineHeight: 1.4
  button:
    fontFamily: Space Grotesk
    fontSize: 16px
    fontWeight: 700
    lineHeight: 1
rounded:
  sm: 8px
  md: 12px
  lg: 16px
  xl: 20px
  full: 9999px
spacing:
  xs: 4px
  sm: 8px
  md: 12px
  lg: 16px
  xl: 20px
  2xl: 24px
  3xl: 32px
  4xl: 40px
  gutter: 20px
  margin-desktop: 40px
components:
  page:
    backgroundColor: "{colors.neutral}"
  text-heading:
    textColor: "{colors.ink}"
    typography: "{typography.h1}"
  text-body:
    textColor: "{colors.ink-muted}"
    typography: "{typography.body-md}"
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.ink}"
    typography: "{typography.button}"
    rounded: "{rounded.lg}"
    height: 50px
  button-secondary:
    backgroundColor: "{colors.secondary}"
    textColor: "#ffffff"
    typography: "{typography.button}"
    rounded: "{rounded.md}"
    height: 44px
  button-ghost:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.secondary}"
    typography: "{typography.caption}"
    rounded: "{rounded.sm}"
    padding: 8px
  card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.lg}"
    padding: "{spacing.lg}"
  card-ticket:
    backgroundColor: "{colors.surface}"
    padding: "{spacing.lg}"
  divider:
    backgroundColor: "{colors.line}"
    size: 1px
  chip-zone-public:
    backgroundColor: "{colors.zone-public}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: 6px
  chip-zone-opd:
    backgroundColor: "{colors.zone-opd}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: 6px
  chip-zone-diagnostic:
    backgroundColor: "{colors.zone-diagnostic}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: 6px
  chip-zone-pharmacy:
    backgroundColor: "{colors.zone-pharmacy}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: 6px
  chip-zone-rehab:
    backgroundColor: "{colors.zone-rehab}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: 6px
  chip-zone-ipd:
    backgroundColor: "{colors.zone-ipd}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: 6px
  chip-zone-support:
    backgroundColor: "{colors.zone-support}"
    textColor: "{colors.ink}"
    typography: "{typography.caption}"
    rounded: "{rounded.full}"
    padding: 6px
  badge-routable:
    backgroundColor: "{colors.success-tint}"
    textColor: "{colors.success}"
    rounded: "{rounded.full}"
  badge-not-routable:
    backgroundColor: "{colors.warning-tint}"
    textColor: "{colors.warning}"
    rounded: "{rounded.full}"
  sheet:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.xl}"
    padding: "{spacing.lg}"
---

# CarePath — DESIGN.md

## Overview

CarePath guides a hospital patient from one visit step to the next and shows
staff the same journey from the operations side. It is not a generic health
app: the product's real subject matter is hospital wayfinding signage — the
colored zone strips, the dark wall linework, and the orange line painted on
the floor that says "follow this to Pharmacy." This design system does not
invent a separate app skin; it extracts that signage system directly from
the project's own SVG floor plans (`packages/floorplans/floors/*.svg`) so the
map a patient looks at and the chrome around it are the same visual
language.

Two registers, one system:

- **Patient** — mobile only, opened from a LINE LIFF webview, often by an
  anxious or elderly person who has never seen the building. One primary
  action per screen, large type, no clinical jargon (a patient never sees
  the word "ServicePoint" — that is staff vocabulary from the domain model).
- **Staff** — mobile-first but expected to scale to a desktop console for a
  front desk or nurse station. Optimized for scanning many rows fast and
  visually matching a list entry to a colored area on the map.

Personality: calm, legible, procedural — closer to airport wayfinding
signage than to a consumer health app. Warm but not soft; dense information
reads like a departmental directory board, not a dashboard SaaS kit.

## Colors

The palette is lifted, value-for-value, from the `:root` custom properties
already defined in `packages/floorplans/floors/*.svg` — this is a
constraint, not a preference, so the app UI never clashes with the maps it
renders.

- **Primary (#ff7a00):** The route line color from the floor plans. Means
  exactly one thing everywhere it appears: "this is your path, do this now."
  Used only for the patient's primary navigate action, the current-step
  marker on the journey rail, and the "needs attention" KPI on the staff
  dashboard. Never used decoratively.
- **Secondary (#0b6e99):** The navigation-node color from the floor plans.
  The staff system's primary interactive color (links, staff buttons, active
  nav item) — kept distinct from primary so staff actions and "your route"
  never compete for attention on the same screen.
- **Ink (#1f2b33) / Ink-muted (#66727d):** Text. Ink-muted matches the
  `--muted` value from the SVGs so map labels and UI captions read as one
  family.
- **Neutral (#f4f6f5):** Page background — the exact corridor-floor gray
  from the SVGs, not a generic warm cream. Surfaces (`#ffffff`) sit on top
  of it as cards/panels.
- **Line (#dde3e3):** Hairline borders and dividers. Cards are mostly
  bordered, not shadowed — see Elevation & Depth.
- **Success / Warning:** Muted, desaturated — `success` marks a place as
  routable, `warning` marks a place that is selectable but has no
  navigation node yet (a real state in the domain: see
  `packages/floorplans/README.md`, e.g. Rehab, IPD). Never a bright
  semantic red/green; this is an operational status, not an alert.
- **Zone colors (`zone-*`):** Copied one-to-one from the floor-plan SVGs'
  zone tints (Public, OPD, Diagnostic/Imaging, Pharmacy, Rehab, IPD,
  Support). Used as small chips in staff lists so a row can be pattern-matched
  to a map area by color alone, without reading text.

## Typography

Two families, each doing a different job — never mixed within the same
piece of text.

- **Noto Sans Thai** carries all prose: headlines, body copy, captions. It
  is also the label face used inside the floor-plan SVGs, so on-screen text
  and map labels feel like one signage system rather than an app bolted on
  top of a map.
- **Space Grotesk** is reserved for numerals and short codes: distances,
  minutes, queue numbers, and place/service codes (`REG-01`, `V-2384`,
  `11 นาที`). Its geometric, slightly technical shapes read like wayfinding
  numerals — deliberately more "signage" than "UI font."

No serif, no monospace. Thai body copy never goes above 16px line length
constraints that would force horizontal scroll on a 360–390px viewport —
patient screens are single-column by design.

## Layout

Mobile-first for both audiences, but the two registers resolve the
mobile → desktop transition differently:

- **Patient:** stays single-column at every width (max content width ~480px,
  centered) because it is a LIFF webview, never a desktop surface. A sticky
  bottom action bar sits in the one-thumb zone; a draggable bottom sheet
  carries turn-by-turn text as an accessible alternative to reading the map.
- **Staff:** a mobile viewport collapses everything to a single column of
  cards with a 4-item bottom tab bar (a front-desk or ward clerk checking a
  phone while walking the floor). At a desktop breakpoint (~1280px+) the
  same content reorganizes into a fixed 220px left nav rail plus a
  multi-column body — tabular lists become real `<table>` elements instead
  of stacked cards, because tabular data deserves a table once there is
  room for one.

Spacing scale is 4px-based (`xs`–`4xl`); page gutters are 20px on mobile,
40px on desktop. Journey-rail and list rows use consistent 12–16px vertical
rhythm rather than ad hoc margins.

## Elevation & Depth

Mostly flat. Cards and rows are separated with the 1px `divider` token
(`colors.line`), not a drop shadow — this keeps the staff board reading like
a directory board,
not a stack of SaaS cards. Shadow is reserved for the two things that are
*actually* floating above other content: the patient's sticky bottom action
bar and the map's bottom sheet, both lifted with a soft upward shadow so
their overlap with scrolling content underneath is legible.

## Shapes

Two shape grammars, deliberately different for the two registers:

- **Patient:** mostly soft (`rounded.lg`/`xl`, pill buttons and pills for
  queue numbers), except the current-step card, which uses an asymmetric
  radius — sharp top-left corner, rounded everywhere else (`4px 16px 16px
  16px`) — a small "ticket stub" cue that this card is the one you're
  holding right now.
- **Staff:** flatter and sharper (`rounded.sm`/`md`), rows and table cells
  stay square; only pills (zone chips, status badges) use `rounded.full`.
  Staff surfaces should feel closer to a printed board than a rounded app
  card.

Node/route markers on the map itself keep the exact shapes used in the SVGs
(filled circles for nodes, rounded-cap orange strokes for the route) so the
in-app schematic map and the real floor-plan asset are visually
interchangeable.

## Components

- **Buttons:** `button-primary` (orange fill, dark ink text — white fails
  WCAG AA against this orange, see Do's and Don'ts) is the single patient
  CTA per screen — never more than one on screen at a time.
  `button-secondary` (blue fill, white text) is for staff primary actions.
  `button-ghost` (white fill, blue outline and text) is for staff secondary
  actions ("ช่วยนำทาง", "ตรวจสอบคิว").
- **Journey rail:** a vertical line of step nodes (filled + check = done,
  large glowing orange = current, outlined = pending) connecting into an
  expanded "ticket" card for the current step and a lightly emphasized card
  for the next step. The line itself changes color ahead of the current
  step to orange, echoing the map's route line.
- **Chips (`chip-zone-*`):** one variant per floor-plan zone (`public`,
  `opd`, `diagnostic`, `pharmacy`, `rehab`, `ipd`, `support`) — a small
  filled dot or pill in that zone's exact color plus a Space Grotesk code
  (`LAB-01`), always with dark `ink` text since every zone tint is pale.
  The atomic unit that ties a staff list row to a place on the map.
- **Status badges:** `badge-routable` (success) / `badge-not-routable`
  (warning) — always paired with a place row, communicating whether a
  `Place` currently has a navigation-graph entry node (per
  `packages/floorplans/README.md`).
- **Bottom sheet:** rounded top corners (`rounded.xl`), drag handle,
  summary line in `display-stat` + `body-sm`, then a numbered steps list.
- **Nav:** 4-item bottom tab bar on staff mobile; fixed left rail (icon +
  label, active state in `secondary-tint`) on staff desktop. Patient has no
  persistent nav chrome beyond the back chevron — the journey is one linear
  flow, not a set of destinations to switch between.
- **Table (staff desktop only):** real `<table>` with a light header row,
  1px row dividers, chip/badge cells — the responsive counterpart to the
  mobile card list, not a separate design.

## Do's and Don'ts

- Do treat `primary` (orange) as meaning "your route, right now" — never use
  it for a decorative accent, a staff action, or more than one element per
  screen.
- Do keep zone chip colors byte-identical to the `:root` values in
  `packages/floorplans/floors/*.svg` — if a zone color ever changes, change
  it in both places together.
- Do use dark `ink` text on `primary` (orange) fills, never white — white on
  this orange measures 2.6:1, well under WCAG AA's 4.5:1 minimum for normal
  text; `ink` on the same orange measures ~5.5:1.
- Don't show patients internal vocabulary (`ServicePoint`, `VisitStep`,
  status enums like `IN_PROGRESS`) — translate every status to plain Thai
  ("กำลังดำเนินการ", "รอดำเนินการ").
- Don't give a `Place` a routable-looking badge or a route line on the map
  unless it actually has a navigation-graph node — surface the "ยังไม่รองรับ
  เส้นทาง" state instead of hiding the room.
- Don't use an ALL-CAPS tracked-out label, a middle-dot-joined meta string,
  or a generic map pin icon — this system has its own numeral face and its
  own node/route iconography; use those instead of default UI chrome.
- Do keep every touch target ≥44px and every patient screen to exactly one
  primary action — this app is used by people who may be anxious, elderly,
  or in a hurry.
- Don't add a drop shadow to an ordinary card; reserve shadow for elements
  that visually float above scrolling content (sticky bar, bottom sheet).
