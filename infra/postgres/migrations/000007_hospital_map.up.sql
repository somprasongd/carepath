-- #24: hospital map model — building → floor → place, the spatial side that
-- service_point.place_id has pointed at as an opaque SVG place id since
-- 000001. Places carry their position on the floor-plan SVG (floor-local
-- coordinates of the PLACE_ENTRY node from packages/floorplans/graphs) so a
-- service point resolves to floor + coordinate + entry node. The walkable
-- nav_node/nav_edge graph itself is #26's slice; entry_node_id stays a plain
-- string reference until then.
--
-- Seeds and the service_point FK share this migration because the FK can only
-- be added once every existing service_point.place_id (REG-01, LAB-01,
-- PHARMACY-01, XRAY-01) has a matching place row.

CREATE TABLE IF NOT EXISTS carepath.building (
    building_id TEXT PRIMARY KEY,
    code        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS carepath.floor (
    floor_id    TEXT PRIMARY KEY,
    building_id TEXT NOT NULL REFERENCES carepath.building (building_id),
    code        TEXT NOT NULL, -- display ordinal, e.g. "1", "2"
    name        TEXT NOT NULL,
    level_order INT    NOT NULL, -- vertical display/sort order
    UNIQUE (building_id, code)
);

CREATE TABLE IF NOT EXISTS carepath.place (
    place_id     TEXT PRIMARY KEY,
    floor_id     TEXT NOT NULL REFERENCES carepath.floor (floor_id),
    name         TEXT NOT NULL,
    place_type   TEXT NOT NULL CHECK (place_type IN
        ('ROOM', 'COUNTER', 'WAITING_AREA', 'RESTROOM', 'ELEVATOR', 'STAIRWAY', 'RAMP', 'ENTRANCE', 'AMENITY')),
    x            NUMERIC, -- floor-local SVG units, see graphs/*.json coordinateSpace
    y            NUMERIC,
    entry_node_id TEXT -- PLACE_ENTRY node in the floor graph; becomes a FK in #26
);

INSERT INTO carepath.building (building_id, code, name)
VALUES ('BLD-I13', 'I-13', 'Building I-13')
ON CONFLICT (building_id) DO NOTHING;

INSERT INTO carepath.floor (floor_id, building_id, code, name, level_order)
VALUES
    ('I-1301', 'BLD-I13', '1', 'Ground Floor', 1),
    ('I-1302', 'BLD-I13', '2', 'Upper Floor', 2)
ON CONFLICT (floor_id) DO NOTHING;

-- Only the places service points map to today; the rest of the plan waits
-- for the graph seed in #26. Coordinates are the PLACE_ENTRY node values, so
-- the destination pin and the future route endpoint are the same point.
INSERT INTO carepath.place (place_id, floor_id, name, place_type, x, y, entry_node_id)
VALUES
    ('REG-01',      'I-1301', 'Reception',         'COUNTER', 150, 190, 'node-reception'),
    ('PHARMACY-01', 'I-1301', 'Pharmacy',          'ROOM',    885, 190, 'node-pharmacy'),
    ('XRAY-01',     'I-1301', 'X-Ray',             'ROOM',    650, 485, 'node-xray'),
    ('OPD-NS-01',   'I-1301', 'OPD Nurse Station', 'ROOM',    445, 330, 'node-opd-ns'),
    ('LAB-01',      'I-1302', 'Blood Collection',  'ROOM',    165, 405, 'node-blood-collection')
ON CONFLICT (place_id) DO NOTHING;

-- The OPD mapping (#24 AC): code DOCTOR matches the HIS serviceCode of the
-- doctor-consult step so the journey projection resolves it; the nurse
-- station is the only OPD place with an entry node so far.
INSERT INTO carepath.service_point (id, code, name, place_id)
VALUES ('SP-OPD', 'DOCTOR', 'OPD', 'OPD-NS-01')
ON CONFLICT (id) DO NOTHING;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'service_point_place_id_fkey'
    ) THEN
        ALTER TABLE carepath.service_point
            ADD CONSTRAINT service_point_place_id_fkey
            FOREIGN KEY (place_id) REFERENCES carepath.place (place_id);
    END IF;
END $$;
