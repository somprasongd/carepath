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
