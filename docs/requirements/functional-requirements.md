# Functional Requirements

## FR-01 Patient entry

The patient can open CarePath from LINE OA / LIFF and establish an application session.

## FR-02 Visit resolution

CarePath can resolve the patient's active visit using the HIS integration or mock HIS.

## FR-03 Journey display

CarePath displays current, completed, and next visit steps.

## FR-04 Next service destination

The next actionable visit step resolves to a `ServicePoint` and physical `Place`.

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

## FR-13 Pathway template management — *Must (M2)*

An administrator can create, edit, and version Care Pathway Templates, each composed of an ordered list of steps with before/after prerequisite constraints between steps.

## FR-14 Pathway-template service-code checklist — *Must (M3)*

Registration/screening staff can select a Care Pathway Template and see its ordered service-code checklist, to enter into the HIS when opening the patient's visit there. Per ADR-0008 the HIS is the sole system of record for visit opening and service ordering — CarePath does not create the visit or write its steps; the projected `VisitStep` list appears only once the HIS reports the corresponding `service.requested` events.

## FR-15 Service-point staff console — *Must (M7)*

Staff at a service point can call the next queue ticket and update a visit step's status (arrived / in progress / completed / skipped).

## FR-16 Unplanned step insertion — *Must (M7)*

Staff can insert an additional, previously unplanned step into a patient's remaining visit steps without violating the existing before/after prerequisite constraints.

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
