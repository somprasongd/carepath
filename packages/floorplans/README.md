# Floor Plans

Source-controlled floor-plan and navigation-graph data for the MVP.

```
floors/
  i-1301-ground.svg   Ground Floor (Floor ID I-1301)
  i-1302-upper.svg    Upper Floor  (Floor ID I-1302)
graphs/
  i-1301.json         Navigation graph for the ground floor
  i-1302.json         Navigation graph for the upper floor
```

Rules:

- SVG is the visual layer.
- Navigation graph is separate JSON/data (ADR-0002).
- Stable `placeId` values connect the SVG, ServicePoint mapping, and navigation entry nodes.

## SVG conventions

| Selector | Meaning |
| --- | --- |
| `id="floor-i-1301"` / `data-floor="I-1301"` | Floor root group. All node/room coordinates below are in this group's local space. |
| `id="room-*"` | Clickable room / area |
| `id="node-*"` | Navigation graph node (matches `nodes[].id` in the graph JSON) |
| `data-place-id="..."` | Stable `placeId` — the join key to ServicePoint and to `nodes[].placeId` |
| `data-zone="..."` | Positioning zone, used to map a Zigbee/coarse location to an area |
| `id="route-layer"` | Layer for drawing the computed route; `opacity="0"` by default |

The plans are schematic for digital navigation and service flow — not construction drawings, and
not to real-world scale. Graph `distance` values are SVG units, not metres.

## Place IDs

Ground floor (I-1301): `REG-01`, `CASHIER-01`, `WAITING-01`, `PHARMACY-01`, `SUPPORT-01`,
`OPD-NS-01`, `OPD-EXAM-01`, `OPD-SUPPORT-01`, `REHAB-01`, `IPD-01`, `XRAY-01`, `CT-01`,
`TRIAGE-01`, `ER-EXAM-04`, `WOUND-01`.

Upper floor (I-1302): `WELLNESS-NS-01`, `PRECOUNSEL-01`, `LAB-01`, `BOF-01`, `LONGEVITY-01`,
`LONGEVITY-02`, `FUTURE-01`, `STAFF-01`.

`LAB-01` is the Blood Collection room on the upper floor — these plans have no separate
laboratory space, so the existing `LAB` service point maps there.

Rooms without a `node-*` (Rehab, IPD, OPD Exam/Support, Support, Future Expansion, Staff) are
selectable but not yet routable; add a node to the SVG and the matching graph JSON to route to them.

## Drawing a route

```js
const svg = document.querySelector('#floor-map')

svg.addEventListener('click', (e) => {
  const room = e.target.closest('[id^="room-"]')
  if (!room) return
  console.log({
    id: room.id,
    placeId: room.dataset.placeId,
    zone: room.dataset.zone ?? room.closest('[data-zone]')?.dataset.zone,
  })
})

function showRoute(svgDoc, points) {
  const routeLayer = svgDoc.getElementById('route-layer')
  routeLayer.querySelector('polyline').setAttribute('points', points.map((p) => `${p.x},${p.y}`).join(' '))
  routeLayer.setAttribute('opacity', '1')
}
```

Each floor graph also carries a `transitions` array linking `node-lift` / `node-stairs` to the
same node on the other floor, so cross-floor routing stays a single graph search. The stairs
transition is marked `accessible: false`.
