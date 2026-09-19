# Technical Blueprint

## Technology baseline

### Frontend

- React
- TypeScript
- Vite
- Tailwind CSS
- shadcn/ui
- TanStack Query
- TanStack Router
- TanStack Table where needed
- React Hook Form + Zod
- Zustand for small client-side state
- LINE LIFF SDK for patient channel
- SVG for floor-plan rendering

### Backend

- Go
- Fiber v3
- REST API
- OpenAPI contracts
- PostgreSQL 17
- Modular Monolith for the CarePath API

### Integration and realtime

- HIS Adapter pattern
- Mock HIS for hackathon/demo
- SSE or WebSocket for live journey/location updates
- QR baseline location provider
- Zigbee integration through MQTT/Zigbee2MQTT and a positioning service when enabled

### Deployment

- Docker Compose for local/hackathon
- Kubernetes-ready boundaries for later production deployment

## Backend modules

```text
identity
visit
journey
servicepoint
hospitalmap
navigation
location
notification
integration
```

Recommended internal shape per module:

```text
module/
├── domain/
├── application/
├── infrastructure/
└── transport/http/
```

The project can progressively adopt ports/adapters without forcing microservices.

## Core runtime chain

```text
Visit
  -> VisitStep / Journey
  -> ServicePoint
  -> Place
  -> NavigationNode
  -> Route
```

## Location abstraction

```text
LocationService
  ├── QRLocationProvider
  ├── ZigbeeLocationProvider
  └── ManualLocationProvider
```

The rest of the domain consumes the normalized result rather than vendor-specific raw data:

```json
{
  "buildingId": "BLDG-A",
  "floorId": "I-1302",
  "zoneId": "WELLNESS",
  "placeId": "LAB-01",
  "confidence": 1.0,
  "source": "QR"
}
```

## Navigation approach

SVG is presentation; routing uses a graph.

- `NavNode`: floor, x, y, type
- `NavEdge`: from, to, distance, accessible, direction
- Algorithm: Dijkstra first; A* is a compatible future optimization

## Floor-plan approach

Floor-plan SVG elements should carry stable IDs/data attributes, for example:

```xml
<rect id="room-blood-collection" data-place-id="LAB-01" />
<rect id="room-pharmacy" data-place-id="PHARMACY-01" />
```

The plans live in `packages/floorplans/` — `floors/*.svg` for the visual layer and `graphs/*.json`
for the matching `NavNode`/`NavEdge` data. See that package's README for the full element/attribute
convention (`room-*`, `node-*`, `data-place-id`, `data-zone`, `route-layer`).

The frontend overlays current position, destination, and the computed route without embedding business flow into the SVG itself.
