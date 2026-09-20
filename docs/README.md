# CarePath Documentation

This directory is the working source of truth for architecture, requirements, decisions, and external-system integration.

## Architecture

- [System Architecture](architecture/system-architecture.md)
- [Technical Blueprint](architecture/technical-blueprint.md) — canonical description of the `apps/api` internal module structure
- [Web App Architecture](architecture/web-app.md) — canonical description of the `apps/web` structure, data flow, and conventions
- [Domain Model](architecture/domain-model.md)
- [Deployment](architecture/deployment.md)

## Requirements

- [Product Requirements](requirements/product-requirements.md)
- [Functional Requirements](requirements/functional-requirements.md)
- [Non-Functional Requirements](requirements/non-functional-requirements.md)
- [MVP Scope](requirements/mvp-scope.md)
- [User Stories](requirements/user-stories.md)
- [Use Case Diagram](requirements/use-case-diagram.md)

## Architecture Decision Records

- [ADR-0001 Monorepo + Modular Monolith](adr/0001-monorepo-modular-monolith.md)
- [ADR-0002 Separate Care Graph and Navigation Graph](adr/0002-separate-care-and-navigation-graphs.md)
- [ADR-0003 SVG Floor Plan for MVP](adr/0003-svg-floor-plan.md)
- [ADR-0004 Location Provider Abstraction](adr/0004-location-provider-abstraction.md)
- [ADR-0005 HIS Adapter and Mock HIS](adr/0005-his-adapter-and-mock-his.md)
- [ADR-0006 REST + OpenAPI Contracts](adr/0006-rest-openapi.md)
- [ADR-0007 Hexagonal Module Layout with Transaction-in-Context](adr/0007-hexagonal-modules-transaction-in-context.md)
- [ADR-0008 Canonical HIS Event/Command Contract](adr/0008-his-canonical-event-contract.md) — superseded in part by ADR-0009
- [ADR-0009 CarePath Owns the Journey Plan](adr/0009-carepath-owns-journey-plan.md)
- [ADR-0010 Staff/Admin Authentication (argon2id + JWT)](adr/0010-staff-auth-jwt-argon2.md)
- [ADR-0011 Visit Share Link (relative tracking token)](adr/0011-visit-share-link.md)
- [ADR-0012 Client-Owned Display Text (i18n)](adr/0012-client-owned-display-text.md)

## Integration

- [Mock HIS](integration/mock-his.md)
- [LINE OA / LIFF](integration/line-liff.md)
- [QR and Zigbee Location](integration/location-zigbee.md)

## APIs

- [CarePath API](api/carepath-api.md)
- [Mock HIS API](api/mock-his-api.md)
- Shared OpenAPI contracts: [`packages/contracts/openapi/`](../packages/contracts/openapi/) — **the source of truth** for externally visible behavior (ADR-0006)
- Live Swagger docs: `apps/api` generates a spec from handler comments with [swaggo/swag](https://github.com/swaggo/swag) (`make swag` to regenerate, output committed in `apps/api/docs/`), served at `GET /swagger` and `GET /api/openapi.json` when the API runs. Treat the generated spec as a developer-facing reference; if it disagrees with the contracts, the contracts win and the handlers should be fixed.

## Designs

- [Patient & Staff UI (mobile-first mockup)](designs/patient-staff-ui.html) — exported interactive design canvas covering the patient journey/navigation screens and the staff dashboard/service-point mapping screens, mobile and desktop. Reference for visual direction only — not wired into `apps/web`.

## Deliverables

Hackathon submission deliverables (brief §10), assembled from the docs above:

- [1. Requirement Specification](deliverables/01-requirement-specification.md)
- [2. ER Diagram and Database Structure](deliverables/02-er-diagram-database.md)
- [3. Prototype / Wireframe](deliverables/03-prototype-wireframe.md)
- 4. Working Software — not a document; see `make up` / `docker compose up --build`
- [5. Test Result](deliverables/05-test-result.md)
- [6. Presentation Slides (outline)](deliverables/06-presentation-slides.md)
- [7. Demo Script (ซ้อมจริงแล้ว)](deliverables/07-demo-script.md)
- [Presentation Outline — Hospital/Stakeholder Review](deliverables/presentation-outline-hospital-review.md) — separate script for the HIS-integration/executive-dashboard presentation round, not the hackathon brief's numbered deliverables

## Process

- [AI-Native SDLC Workflow](process/ai-native-sdlc-workflow.md) — how intent → design → plan → build → test → deploy is meant to flow in this repo, and what's still missing (CI, tests, hooks/skills).

## Documentation rules

1. Requirements describe **what** the system must do.
2. Architecture docs describe **how the system is organized**.
3. ADRs capture **why a major technical decision was made**.
4. API contracts define externally visible behavior and are version-controlled.
5. A real HIS should be integrated through an adapter; business logic must not depend on a vendor-specific HIS API.
