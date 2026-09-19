# CarePath Documentation

This directory is the working source of truth for architecture, requirements, decisions, and external-system integration.

## Architecture

- [System Architecture](architecture/system-architecture.md)
- [Technical Blueprint](architecture/technical-blueprint.md)
- [Domain Model](architecture/domain-model.md)
- [Deployment](architecture/deployment.md)

## Requirements

- [Product Requirements](requirements/product-requirements.md)
- [Functional Requirements](requirements/functional-requirements.md)
- [Non-Functional Requirements](requirements/non-functional-requirements.md)
- [MVP Scope](requirements/mvp-scope.md)
- [Initial User Stories](requirements/user-stories.md)

## Architecture Decision Records

- [ADR-0001 Monorepo + Modular Monolith](adr/0001-monorepo-modular-monolith.md)
- [ADR-0002 Separate Care Graph and Navigation Graph](adr/0002-separate-care-and-navigation-graphs.md)
- [ADR-0003 SVG Floor Plan for MVP](adr/0003-svg-floor-plan.md)
- [ADR-0004 Location Provider Abstraction](adr/0004-location-provider-abstraction.md)
- [ADR-0005 HIS Adapter and Mock HIS](adr/0005-his-adapter-and-mock-his.md)
- [ADR-0006 REST + OpenAPI Contracts](adr/0006-rest-openapi.md)

## Integration

- [Mock HIS](integration/mock-his.md)
- [LINE OA / LIFF](integration/line-liff.md)
- [QR and Zigbee Location](integration/location-zigbee.md)

## APIs

- [CarePath API](api/carepath-api.md)
- [Mock HIS API](api/mock-his-api.md)
- Shared OpenAPI contracts: [`packages/contracts/openapi/`](../packages/contracts/openapi/)

## Designs

- [Patient & Staff UI (mobile-first mockup)](designs/patient-staff-ui.html) — exported interactive design canvas covering the patient journey/navigation screens and the staff dashboard/service-point mapping screens, mobile and desktop. Reference for visual direction only — not wired into `apps/web`.

## Process

- [AI-Native SDLC Workflow](process/ai-native-sdlc-workflow.md) — how intent → design → plan → build → test → deploy is meant to flow in this repo, and what's still missing (CI, tests, hooks/skills).

## Documentation rules

1. Requirements describe **what** the system must do.
2. Architecture docs describe **how the system is organized**.
3. ADRs capture **why a major technical decision was made**.
4. API contracts define externally visible behavior and are version-controlled.
5. A real HIS should be integrated through an adapter; business logic must not depend on a vendor-specific HIS API.
