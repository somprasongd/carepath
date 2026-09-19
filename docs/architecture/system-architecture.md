# System Architecture

## Objective

CarePath helps a patient understand **what to do next**, **where the next service is**, and **how to get there** inside a hospital. It does not replace the HIS.

```mermaid
flowchart TB
    subgraph CHANNELS[Patient and Staff Channels]
      LINE[LINE OA]
      LIFF[CarePath LIFF / Web]
      ADMIN[Staff / Admin Web]
      LINE --> LIFF
    end

    subgraph EDGE[Application Edge]
      API[CarePath API]
      RT[Realtime SSE/WebSocket]
    end

    LIFF --> API
    ADMIN --> API
    API --> AUTH
    LIFF --> RT

    subgraph CORE[CarePath Core]
      AUTH[Auth / RBAC]
      VISIT[Visit]
      JOURNEY[Journey / Care Graph]
      SP[ServicePoint]
      MAP[Hospital Map]
      NAV[Navigation]
      LOC[Location]
      NOTICE[Notification]
    end

    API --> VISIT
    VISIT --> JOURNEY
    JOURNEY --> SP
    SP --> MAP
    MAP --> NAV
    LOC --> NAV
    API --> NOTICE

    subgraph DATA[Data]
      PG[(PostgreSQL)]
      SVG[SVG Floor Plans]
      GRAPH[Navigation Graph]
    end

    CORE --> PG
    MAP --> SVG
    NAV --> GRAPH

    subgraph HIS[Hospital Systems]
      ADAPTER[HIS Integration Adapter]
      REALHIS[Real HIS / Queue / Appointment]
      MOCK[Mock HIS]
    end

    API --> ADAPTER
    ADAPTER --> REALHIS
    ADAPTER --> MOCK

    subgraph LOCATION[Location Sources]
      QR[QR Location]
      ZB[Zigbee Positioning]
    end

    QR --> LOC
    ZB --> LOC
```

## Responsibility split

| Component | Responsibility |
|---|---|
| HIS | Clinical/transaction source of truth: appointment, visit, orders, queue/service state |
| Care Graph | Determines the current and next care/service step |
| ServicePoint | Maps a logical service to a physical place |
| Hospital Map | Stores building/floor/zone/place relationships |
| Navigation Graph | Calculates a route through walkable nodes and edges |
| Location Provider | Resolves the patient's current place/zone |
| LINE LIFF/Web | Presents journey and navigation |
| Auth | Authenticates staff/admin users (argon2id password → JWT access + rotating refresh token) and carries their roles into every request; patients authenticate separately via LINE ([ADR-0010](../adr/0010-staff-auth-jwt-argon2.md)) |

## Key principle

A care step should not contain route geometry. It references a `ServicePoint`, which resolves to a `Place`, which is connected to the navigation graph. This separation allows service workflow and hospital map/navigation to evolve independently.
