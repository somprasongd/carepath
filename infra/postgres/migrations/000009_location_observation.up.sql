-- #30: location observations — the normalized current-location records that
-- location providers resolve onto (ADR-0004). Append-only per visit; the
-- latest row is the visit's current location. node_id is FK'd to nav_node
-- (seeded by 000008) so a recorded fix always names a canonical graph node
-- "<floorId>/<localId>". confidence stays NULL for exact sources (QR,
-- manual) and is set for probabilistic ones (Zigbee).

CREATE TABLE IF NOT EXISTS carepath.location_observation (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    visit_id    TEXT NOT NULL,                    -- opaque HIS visit reference
    source      TEXT NOT NULL,                    -- QR | ZIGBEE | MANUAL (ADR-0004)
    node_id     TEXT NOT NULL REFERENCES carepath.nav_node (node_id),
    floor_id    TEXT NOT NULL,
    zone        TEXT,                             -- from the graph node when the provider had none
    confidence  DOUBLE PRECISION,                 -- NULL = exact fix
    observed_at TIMESTAMPTZ NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Current-location lookup: newest row of one visit.
CREATE INDEX IF NOT EXISTS location_observation_visit_idx
    ON carepath.location_observation (visit_id, id DESC);
