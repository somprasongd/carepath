# Functional Requirements

## FR-01 Patient entry

The patient can open CarePath from LINE OA / LIFF and establish an application session.

## FR-02 Visit resolution

CarePath can resolve the patient's active visit using the HIS integration or mock HIS.

## FR-03 Journey display

CarePath displays completed, actionable, and pending visit steps. More than one step may be actionable at once — see FR-30.

## FR-04 Next service destination

Every actionable visit step resolves to a `ServicePoint` and physical `Place`; CarePath recommends one of them (e.g. nearest, shortest queue) as the single primary action.

## FR-05 Floor-plan display

CarePath can display the correct building/floor SVG containing the destination.

## FR-06 Current location

CarePath can resolve current location from at least QR. The architecture must allow additional providers.

## FR-07 Route calculation

Given a start node and destination node, CarePath calculates a valid route using the navigation graph.

## FR-08 Route visualization

The route is rendered as an overlay on the SVG floor plan.

## FR-09 Mock HIS

A standalone mock HIS provides deterministic demo data for appointments/visits/service steps without requiring access to the real HIS.

## FR-10 HIS adapter

CarePath core uses a stable internal contract and must not depend directly on vendor-specific HIS endpoints or tables.

## FR-11 Admin configuration

Staff can eventually configure buildings, floors, places, service points, and their mappings. Full CRUD UI is not required for the initial hackathon.

## FR-12 Optional realtime location

When Zigbee is enabled, incoming observations can update the normalized current location without changing journey-domain code.

## FR-13 Journey planning rules — *Must (M2)* — superseded in scope by ADR-0009

An administrator can review the ordering rules the journey planner applies (registration always first; pre-visit diagnostics before the assigned clinic; a clinic-ordered diagnostic implies a return to that clinic unless the encounter is confirmed closed; cashier once per visit; pharmacy after cashier when a drug was prescribed — see ADR-0009 §3–§6). Per ADR-0009 these rules are hospital policy encoded in the planner, not a per-patient template staff assemble: the HIS reports no ordered step list for CarePath to template against. What FR-13 originally called a "Care Pathway Template" is retired.

## FR-14 Journey visibility for registration staff — *Must (M3)*

Registration/screening staff can look up a visit by VN and see the plan CarePath's journey planner derived for it. Per ADR-0009 the HIS is the sole system of record for opening a visit and assigning clinics/orders — CarePath does not create the visit; the plan appears only once the HIS reports the corresponding facts (`visit.opened`, `order.placed`, ...).

## FR-15 Service-point staff console — *Must (M7)*

Staff at a service point can call the next queue ticket and update a visit step's status (arrived / in progress / completed / skipped).

## FR-16 Unplanned step insertion — *Must (M7)*

An order placed mid-visit (e.g. a doctor ordering an extra test) is automatically reflected in the patient's plan by the journey planner (FR-28) — staff do not manually insert a step. Staff retain one manual action: confirming whether the patient returns to the ordering clinic afterward, when the planner's inference (FR-29) needs an override or the HIS cannot report `encounter.completed`.

## FR-17 Queue and wait-time estimate — *Should (S1)*

CarePath displays the current queue length at a service point and an estimated wait time derived from the number of outstanding tickets.

## FR-18 Authentication and role-based access control — *Must (M8)*

Users authenticate before using staff/admin/executive functions. Access to data and actions is restricted by role; a patient can see only their own visit data.

## FR-19 Multi-language support — *Should (S4)*

Patient-facing screens and navigation instructions are available in Thai and English, switchable at runtime.

## FR-20 Accessibility navigation options — *Should (S5)*

Patient-facing screens support a large-text display mode and a route option that avoids stairs for wheelchair users.

## FR-21 Queue-proximity notification — *Should (S6)*

CarePath notifies a patient when their queue position is approaching, so they do not need to wait directly outside the service point.

## FR-22 Executive bottleneck dashboard — *Should (S7)*

A hospital executive can view average wait time per service point, identify bottleneck service points, and see the average total time patients spend in the hospital.

## FR-23 Automatic re-sequencing (stretch) — *Could (C1)*

When a service point's queue is abnormally long, CarePath may propose reordering a patient's remaining steps, without violating prerequisite constraints.

## FR-24 Relative tracking link (stretch) — *Could (C2)*

A relative can follow a patient's current step from their own device via a time-limited shareable link.

## FR-25 Voice-guided navigation (stretch) — *Could (C3)*

Navigation instructions can be read aloud as speech.

## FR-26 Nearby amenities (stretch) — *Could (C4)*

CarePath can suggest a nearby waiting area, restroom, or food stall along the route to the next step.

## FR-27 Automatic journey plan derivation — *Must (M2/M3)* — added by ADR-0009

CarePath derives a patient's ordered visit plan from HIS-reported facts (visit opened, clinic assignment, orders, encounter completion) rather than from a step list the HIS provides directly, applying the ordering rules in ADR-0009 §3: registration first; pre-visit diagnostics before the assigned clinic; cashier exactly once per visit, regardless of clinic count; pharmacy after cashier only when a drug was ordered.

## FR-28 Plan recomputation on new facts — *Must (M7)* — added by ADR-0009

Every inbound fact (a new order, an order's result, an encounter closing) triggers CarePath to recompute the plan and reconcile it against what is already in progress: a step already `STARTED`, `COMPLETED`, or `CANCELLED` is never removed or reordered; a step still `PENDING`/`WAITING` may be added, removed, or reordered as the facts change.

## FR-29 Return-to-clinic inference — *Must (M7)* — added by ADR-0009

When a diagnostic order (lab, x-ray, EKG, ultrasound) is placed while a clinic's encounter with the patient is still open, CarePath infers the patient will return to that same clinic afterward and adds a follow-up step; a drug order never implies a return. The clinic confirming the encounter is finished (via the HIS fact or the staff override) drops any not-yet-started follow-up step for that round.

## FR-30 Concurrent actionable steps — *Must (M7)* — added by ADR-0009

Steps that do not depend on each other (e.g. a lab test and an X-ray both ordered before the same clinic visit) are all actionable at once; CarePath does not force an arbitrary order between them, and recommends one for the patient's primary action (FR-04).
