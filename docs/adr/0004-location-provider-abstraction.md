# ADR-0004: Location Provider Abstraction

- Status: Accepted
- Date: 2026-09-19

## Decision

Expose a normalized location interface and support multiple providers.

Initial providers:

1. QR location - MVP baseline.
2. Zigbee location - optional/phase 2.
3. Manual selection - fallback/debug.

## Consequences

CarePath journey/navigation logic consumes a normalized place/zone result and does not depend on Zigbee protocol details.
