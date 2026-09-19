# CarePath Monorepo

CarePath is a patient journey and indoor navigation platform for hospitals. It sits above the HIS and combines care/service flow, hospital spatial data, indoor routing, and current-location providers such as QR or Zigbee.

## Core idea

- **HIS** knows what service the patient needs.
- **Care Graph** knows what the next step is.
- **ServicePoint** links a care step to a physical place.
- **Hospital Map** knows where that place is.
- **Navigation Graph** knows how to get there.
- **Location Provider** knows where the patient is now.
- **LINE LIFF/Web** presents the journey and route to the patient.

## Repository layout

```text
carepath-monorepo/
├── apps/
│   ├── api/              # CarePath API - Go + Fiber v3
│   ├── mock-his/         # Mock HIS service - Go + Fiber v3
│   └── web/              # Patient/Admin web - React + TypeScript + Vite
├── packages/
│   ├── contracts/        # Shared OpenAPI/schema/contracts
│   └── floorplans/       # SVG floor plans (I-1301, I-1302) + navigation graphs
├── infra/
│   ├── docker/           # Dockerfiles
│   └── postgres/         # Local DB bootstrap
├── docs/
│   ├── adr/              # Architecture Decision Records
│   ├── architecture/     # System architecture and technical blueprint
│   ├── requirements/     # Product/functional/NFR/MVP requirements
│   ├── integration/      # Mock HIS, LINE, Zigbee/location integration
│   └── api/              # API behavior and contract notes
├── docker-compose.yml
├── go.work
├── Makefile
└── .env.example
```

## Quick start

### 1. Run with Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

Expected services:

- CarePath Web: `http://localhost:5173`
- CarePath API: `http://localhost:8080`
- Mock HIS: `http://localhost:8090`
- PostgreSQL: `localhost:5432`

### 2. Local Go services

```bash
go work sync
cd apps/mock-his && go run ./cmd/server
cd apps/api && go run ./cmd/server
```

### 3. Local web

```bash
cd apps/web
npm install
npm run dev
```

## Important architecture boundaries

1. CarePath does **not** read the HIS database directly.
2. The HIS integration is hidden behind an adapter/port.
3. Care Graph and Navigation Graph are separate models.
4. Floor plans are SVG-based for the MVP.
5. QR is the baseline location provider; Zigbee is an optional provider through the same abstraction.
6. Mock HIS is intentionally replaceable by a real HIS adapter.
7. MVP does not depend on 3D or realtime Zigbee positioning.

See [`docs/README.md`](docs/README.md) for the documentation index.
