import { zones, type Zone } from './tokens'

export type MapRoom = {
  id: string
  zone: Zone
  label: string
  /** Optional place code, drawn in the signage numeral face. */
  code?: string
  x: number
  y: number
  width: number
  height: number
  /** The room the patient is heading to — drawn with a heavier edge. */
  destination?: boolean
}

export type SchematicMapProps = {
  rooms: MapRoom[]
  corridor: { x: number; y: number; width: number; height: number; label: string }
  /** Route polyline in the same `d` grammar as the floor-plan SVGs. */
  route: string
  /** End-of-route marker. */
  routeEnd: { x: number; y: number }
  /** Current location from the active location provider (ADR-0004). */
  you: { x: number; y: number; label: string }
  viewBox?: string
}

/**
 * The in-app schematic. Node and route marks keep the exact shapes used in
 * packages/floorplans/floors/*.svg — filled circles for nodes, rounded-cap
 * orange strokes for the route — so map and app read as one system.
 */
export function SchematicMap({
  rooms,
  corridor,
  route,
  routeEnd,
  you,
  viewBox = '0 0 340 190',
}: SchematicMapProps) {
  return (
    <svg
      className="cp-map"
      viewBox={viewBox}
      role="img"
      aria-label={`ผังเส้นทางจาก${you.label}ไปยังจุดหมาย`}
    >
      {rooms.map((room) => {
        const zone = zones[room.zone]
        return (
          <g key={room.id}>
            <rect
              x={room.x}
              y={room.y}
              width={room.width}
              height={room.height}
              rx={6}
              fill={zone.fill}
              stroke={zone.edge}
              strokeWidth={room.destination ? 1.5 : 1}
            />
            <text
              x={room.x + room.width / 2}
              y={room.y + room.height / 2 + (room.code ? 0 : 4)}
              textAnchor="middle"
              fontFamily="Noto Sans Thai"
              fontSize={10}
              fontWeight={700}
              fill="var(--cp-ink)"
            >
              {room.label}
            </text>
            {room.code && (
              <text
                x={room.x + room.width / 2}
                y={room.y + room.height / 2 + 14}
                textAnchor="middle"
                fontFamily="Space Grotesk"
                fontSize={8}
                fill="var(--cp-ink-muted)"
              >
                {room.code}
              </text>
            )}
          </g>
        )
      })}

      <rect
        x={corridor.x}
        y={corridor.y}
        width={corridor.width}
        height={corridor.height}
        fill="var(--cp-neutral)"
        stroke="var(--cp-zone-support-edge)"
        strokeWidth={1}
      />
      <text
        x={corridor.x + corridor.width / 2}
        y={corridor.y + corridor.height + 30}
        textAnchor="middle"
        fontFamily="Noto Sans Thai"
        fontSize={9}
        fontWeight={600}
        fill="var(--cp-ink-muted)"
      >
        {corridor.label}
      </text>

      <path
        d={route}
        fill="none"
        stroke="var(--cp-primary)"
        strokeWidth={6}
        strokeLinecap="round"
        strokeLinejoin="round"
        opacity={0.92}
      />
      <circle
        cx={routeEnd.x}
        cy={routeEnd.y}
        r={5}
        fill="var(--cp-primary)"
        stroke="var(--cp-surface)"
        strokeWidth={2}
      />

      <circle cx={you.x} cy={you.y} r={15} fill="var(--cp-secondary)" opacity={0.15} />
      <circle
        cx={you.x}
        cy={you.y}
        r={7}
        fill="var(--cp-secondary)"
        stroke="var(--cp-surface)"
        strokeWidth={3}
      />
      <text
        x={you.x}
        y={you.y + 30}
        textAnchor="middle"
        fontFamily="Noto Sans Thai"
        fontSize={9}
        fontWeight={700}
        fill="var(--cp-secondary)"
      >
        {you.label}
      </text>
    </svg>
  )
}
