# ADR-0005: HIS Adapter and Mock HIS

- Status: Accepted
- Date: 2026-09-19

## Context

A hackathon/demo must work independently of production HIS access while still modeling how production integration will occur.

## Decision

CarePath integrates through an internal HIS port/adapter. A separate Mock HIS implements the same conceptual contract for development and demo.

CarePath must not query the HIS database directly.

## Consequences

- Demo scenarios are deterministic.
- Integration code is isolated from domain logic.
- A real HIS connector can later replace Mock HIS without rewriting journey and navigation modules.
