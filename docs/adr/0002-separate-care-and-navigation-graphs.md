# ADR-0002: Separate Care Graph and Navigation Graph

- Status: Accepted
- Date: 2026-09-19

## Context

A patient's care flow changes for clinical/operational reasons, while hospital walking paths change for spatial/facility reasons. Coupling both models would make changes risky.

## Decision

Maintain separate models:

- **Care Graph / Visit Steps**: what the patient must do next.
- **Navigation Graph**: how to physically reach a place.

Bridge them through `ServicePoint -> Place`.

## Consequences

A pharmacy service step can be remapped to another physical pharmacy without rewriting the care flow. Navigation can also be redesigned without altering clinical/service workflow definitions.
