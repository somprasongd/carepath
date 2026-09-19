# ADR-0003: SVG Floor Plan for MVP

- Status: Accepted
- Date: 2026-09-19

## Decision

Use simplified 2D SVG floor plans for the MVP. Render navigation paths as overlays.

## Rationale

SVG is browser-native, scalable, easy to style, supports stable element IDs, works well with React, and is much less costly than requiring a 3D engine.

## Consequences

3D may be added later using the same `Place` and navigation data, but 3D is not a dependency of the core architecture or MVP.
