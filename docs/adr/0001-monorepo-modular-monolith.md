# ADR-0001: Monorepo and Modular Monolith

- Status: Accepted
- Date: 2026-09-19

## Context

CarePath requires a patient web app, backend API, mock HIS, shared contracts, infrastructure, and documentation. The hackathon needs fast coordination and a simple local development experience.

## Decision

Use one monorepo. Keep the CarePath backend as a modular monolith rather than creating multiple core-domain microservices.

## Consequences

### Positive
- One versioned change can update UI, API, mock HIS, contract, and docs together.
- Lower deployment and debugging complexity.
- Module boundaries can still be explicit.

### Trade-offs
- Teams must maintain import/dependency discipline.
- If a module later requires independent scaling or ownership, extraction will be a separate architecture decision.
