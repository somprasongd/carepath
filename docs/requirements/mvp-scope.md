# MVP Scope

Mirrors the MoSCoW prioritization in the hackathon brief (§5). See [Functional Requirements](functional-requirements.md) for the FR-numbered detail behind each item and [User Stories](user-stories.md) for the per-role stories.

## Must Have — required to pass the baseline bar

- LINE OA entry point, LIFF/web patient experience (M4)
- Hospital map data: buildings, floors, places/service points, connecting routes (M1)
- Care Pathway Template management (M2)
- Patient registration for the day, with pathway template assignment and auto-generated visit steps (M3)
- Patient journey screen: all steps in order with status, and what's next (M4)
- Navigation from current location to next destination, as step-by-step instructions with distance/time estimate (M5)
- Shortest-route calculation from the navigation graph — no hardcoded routes (M6)
- Service-point staff console: call queue, update step status, insert unplanned steps (M7)
- Authentication and role-based access control (M8) — staff/admin username + password login (argon2id), JWT access + refresh tokens, `STAFF`/`ADMIN` roles guarding the staff endpoints ([ADR-0010](../adr/0010-staff-auth-jwt-argon2.md))
- Mock HIS service and a replaceable HIS adapter interface

## Should Have — after Must Have works

- Current queue length and wait-time estimate per service point (S1)
- QR-code current-location scanning (S2)
- Floor plan image with the route drawn on it (S3)
- Thai/English language switch (S4)
- Accessibility mode: large text, stairs-avoiding route for wheelchair users (S5)
- Queue-proximity notification (S6)
- Executive summary: average wait time per service point and bottleneck identification (S7)

## Could Have — if time remains

- Automatic re-sequencing when a service point's queue is abnormally long (C1)
- Relative tracking via a time-limited link (C2)
- Voice-guided navigation instructions (C3)
- Nearby amenity suggestions along the route (C4)

## Optional demonstration

- Realtime location event updates
- Zigbee positioning proof of concept

## Explicitly not required for MVP

- Production HIS integration
- Full realtime queue integration
- 3D floor plan
- Production-grade Zigbee location precision
- Complex multi-building routing
- Full hospital map administration UI (CRUD screens) — data can be seeded/edited without a polished admin UI
- User-management screens (create/disable a user, reset a password) — MVP users are seeded by migration and managed in the database (ADR-0010)
- Password reset/change flows, login rate limiting, MFA, and SSO / hospital Active Directory integration

The architecture should allow all of the above later without making them dependencies of the MVP.
