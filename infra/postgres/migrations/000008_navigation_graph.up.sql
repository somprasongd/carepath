-- #26: the walkable navigation graph — nav_node/nav_edge, separate from the
-- SVG visual layer (ADR-0002). Source of truth is the hand-authored
-- packages/floorplans/graphs/*.json (coordinates and distances are authored
-- data, never computed from SVG geometry); the INSERTs below were generated
-- from that JSON by script, with node ids prefixed by floor because the JSON
-- and SVG ids are floor-local (node-lift / node-stairs exist on both floors).
--
-- Edges are directed per the planned DDL: every JSON connection (same-floor
-- edge or cross-floor transition) is seeded as both A→B and B→A; mirrored
-- transitions declared on both floors collapse via ON CONFLICT. distance is
-- in schematic SVG units (floorplans README), not metres.
--
-- Also completes what 000007 deferred: seeds the remaining places the graph
-- references (names from the floor-plan SVGs) and turns place.entry_node_id
-- into the promised FK — the service-point → destination → entry-node chain
-- ends in a real graph node.

CREATE TABLE IF NOT EXISTS carepath.nav_node (
    node_id   TEXT PRIMARY KEY,              -- <floorId>/<localId>, e.g. I-1301/node-reception
    floor_id  TEXT NOT NULL REFERENCES carepath.floor (floor_id),
    x         NUMERIC NOT NULL,              -- floor-local SVG units of the graph JSON
    y         NUMERIC NOT NULL,
    node_type TEXT NOT NULL CHECK (node_type IN
        ('ENTRANCE', 'PLACE_ENTRY', 'CORRIDOR', 'ELEVATOR', 'STAIRS')),
    zone      TEXT                           -- positioning zone from the graph JSON (PUBLIC, OPD, ER, ...)
);

CREATE TABLE IF NOT EXISTS carepath.nav_edge (
    edge_id      TEXT PRIMARY KEY,           -- <fromNodeId>><toNodeId>
    from_node_id TEXT NOT NULL REFERENCES carepath.nav_node (node_id),
    to_node_id   TEXT NOT NULL REFERENCES carepath.nav_node (node_id),
    edge_type    TEXT NOT NULL CHECK (edge_type IN ('CORRIDOR', 'ELEVATOR', 'STAIRS')),
    distance     NUMERIC NOT NULL,           -- cost: authored SVG units, not metres
    accessible   BOOLEAN NOT NULL DEFAULT true, -- false for stairs; routing avoids them
    CHECK (from_node_id <> to_node_id)
);

CREATE INDEX IF NOT EXISTS nav_node_floor_idx ON carepath.nav_node (floor_id);
CREATE INDEX IF NOT EXISTS nav_edge_from_node_idx ON carepath.nav_edge (from_node_id);
CREATE INDEX IF NOT EXISTS nav_edge_to_node_idx ON carepath.nav_edge (to_node_id);

-- The remaining places the graph references (names/place types from the
-- floor-plan SVGs); x/y are the PLACE_ENTRY node coordinates, same convention
-- as the five places seeded in 000007. Rooms without a graph node (Rehab,
-- IPD, ...) stay unseeded — selectable but not yet routable.
INSERT INTO carepath.place (place_id, floor_id, name, place_type, x, y, entry_node_id)
VALUES
    ('CASHIER-01',     'I-1301', 'Cashier',                 'COUNTER',      390, 190, 'I-1301/node-cashier'),
    ('WAITING-01',     'I-1301', 'Waiting',                 'WAITING_AREA', 620, 190, 'I-1301/node-waiting'),
    ('CT-01',          'I-1301', 'CT',                      'ROOM',         760, 555, 'I-1301/node-ct'),
    ('TRIAGE-01',      'I-1301', 'Triage',                  'ROOM',         965, 330, 'I-1301/node-triage'),
    ('ER-EXAM-04',     'I-1301', 'Exam 4',                  'ROOM',         965, 480, 'I-1301/node-er-exam4'),
    ('WOUND-01',       'I-1301', 'Wound Care',              'ROOM',         965, 555, 'I-1301/node-wound-care'),
    ('WELLNESS-NS-01', 'I-1302', 'Wellness Nurse Station',  'ROOM',         375, 295, 'I-1302/node-wellness-ns'),
    ('PRECOUNSEL-01',  'I-1302', 'Pre-Counselling',         'ROOM',         322, 205, 'I-1302/node-pre-counselling'),
    ('BOF-01',         'I-1302', 'BOF Report',              'ROOM',         322, 405, 'I-1302/node-bof-report'),
    ('LONGEVITY-01',   'I-1302', 'Longevity',               'ROOM',         718, 295, 'I-1302/node-longevity')
ON CONFLICT (place_id) DO NOTHING;

-- The five places seeded in 000007 carry floor-local entry node ids; move
-- them onto the same floor-prefixed global ids the graph uses.
UPDATE carepath.place
SET entry_node_id = floor_id || '/' || entry_node_id
WHERE entry_node_id IS NOT NULL AND entry_node_id NOT LIKE '%/%';

-- nav_node
INSERT INTO carepath.nav_node (node_id, floor_id, x, y, node_type, zone) VALUES
    ('I-1301/node-main-entrance', 'I-1301', 150, -10, 'ENTRANCE', 'PUBLIC'),
    ('I-1301/node-opd-entrance', 'I-1301', 0, 212, 'ENTRANCE', 'OPD'),
    ('I-1301/node-er-entrance', 'I-1301', 1290, 212, 'ENTRANCE', 'ER'),
    ('I-1301/node-reception', 'I-1301', 150, 190, 'PLACE_ENTRY', 'PUBLIC'),
    ('I-1301/node-ramp', 'I-1301', 318, 190, 'CORRIDOR', 'PUBLIC'),
    ('I-1301/node-cashier', 'I-1301', 390, 190, 'PLACE_ENTRY', 'PUBLIC'),
    ('I-1301/node-lift', 'I-1301', 450, 205, 'ELEVATOR', NULL),
    ('I-1301/node-waiting', 'I-1301', 620, 190, 'PLACE_ENTRY', 'PUBLIC'),
    ('I-1301/node-pharmacy', 'I-1301', 885, 190, 'PLACE_ENTRY', 'PHARMACY'),
    ('I-1301/node-stairs', 'I-1301', 962, 205, 'STAIRS', NULL),
    ('I-1301/node-opd-ns', 'I-1301', 445, 330, 'PLACE_ENTRY', 'OPD'),
    ('I-1301/node-xray', 'I-1301', 650, 485, 'PLACE_ENTRY', 'IMAGING'),
    ('I-1301/node-ct', 'I-1301', 760, 555, 'PLACE_ENTRY', 'IMAGING'),
    ('I-1301/node-triage', 'I-1301', 965, 330, 'PLACE_ENTRY', 'ER'),
    ('I-1301/node-er-exam4', 'I-1301', 965, 480, 'PLACE_ENTRY', 'ER'),
    ('I-1301/node-wound-care', 'I-1301', 965, 555, 'PLACE_ENTRY', 'ER'),
    ('I-1302/node-lift', 'I-1302', 480, 295, 'ELEVATOR', NULL),
    ('I-1302/node-stairs', 'I-1302', 955, 295, 'STAIRS', NULL),
    ('I-1302/node-wellness-ns', 'I-1302', 375, 295, 'PLACE_ENTRY', 'WELLNESS'),
    ('I-1302/node-pre-counselling', 'I-1302', 322, 205, 'PLACE_ENTRY', 'WELLNESS'),
    ('I-1302/node-blood-collection', 'I-1302', 165, 405, 'PLACE_ENTRY', 'WELLNESS'),
    ('I-1302/node-bof-report', 'I-1302', 322, 405, 'PLACE_ENTRY', 'WELLNESS'),
    ('I-1302/node-longevity', 'I-1302', 718, 295, 'PLACE_ENTRY', 'LONGEVITY') ON CONFLICT (node_id) DO NOTHING;

-- nav_edge (each JSON connection seeded as both directions;
-- mirrored transitions declared on both floors collapse on conflict)
INSERT INTO carepath.nav_edge (edge_id, from_node_id, to_node_id, edge_type, distance, accessible) VALUES
    ('I-1301/node-main-entrance>I-1301/node-reception', 'I-1301/node-main-entrance', 'I-1301/node-reception', 'CORRIDOR', 200, true),
    ('I-1301/node-reception>I-1301/node-main-entrance', 'I-1301/node-reception', 'I-1301/node-main-entrance', 'CORRIDOR', 200, true),
    ('I-1301/node-opd-entrance>I-1301/node-reception', 'I-1301/node-opd-entrance', 'I-1301/node-reception', 'CORRIDOR', 172, true),
    ('I-1301/node-reception>I-1301/node-opd-entrance', 'I-1301/node-reception', 'I-1301/node-opd-entrance', 'CORRIDOR', 172, true),
    ('I-1301/node-reception>I-1301/node-ramp', 'I-1301/node-reception', 'I-1301/node-ramp', 'CORRIDOR', 168, true),
    ('I-1301/node-ramp>I-1301/node-reception', 'I-1301/node-ramp', 'I-1301/node-reception', 'CORRIDOR', 168, true),
    ('I-1301/node-ramp>I-1301/node-cashier', 'I-1301/node-ramp', 'I-1301/node-cashier', 'CORRIDOR', 72, true),
    ('I-1301/node-cashier>I-1301/node-ramp', 'I-1301/node-cashier', 'I-1301/node-ramp', 'CORRIDOR', 72, true),
    ('I-1301/node-cashier>I-1301/node-lift', 'I-1301/node-cashier', 'I-1301/node-lift', 'CORRIDOR', 75, true),
    ('I-1301/node-lift>I-1301/node-cashier', 'I-1301/node-lift', 'I-1301/node-cashier', 'CORRIDOR', 75, true),
    ('I-1301/node-lift>I-1301/node-waiting', 'I-1301/node-lift', 'I-1301/node-waiting', 'CORRIDOR', 185, true),
    ('I-1301/node-waiting>I-1301/node-lift', 'I-1301/node-waiting', 'I-1301/node-lift', 'CORRIDOR', 185, true),
    ('I-1301/node-waiting>I-1301/node-pharmacy', 'I-1301/node-waiting', 'I-1301/node-pharmacy', 'CORRIDOR', 265, true),
    ('I-1301/node-pharmacy>I-1301/node-waiting', 'I-1301/node-pharmacy', 'I-1301/node-waiting', 'CORRIDOR', 265, true),
    ('I-1301/node-pharmacy>I-1301/node-stairs', 'I-1301/node-pharmacy', 'I-1301/node-stairs', 'CORRIDOR', 92, true),
    ('I-1301/node-stairs>I-1301/node-pharmacy', 'I-1301/node-stairs', 'I-1301/node-pharmacy', 'CORRIDOR', 92, true),
    ('I-1301/node-stairs>I-1301/node-er-entrance', 'I-1301/node-stairs', 'I-1301/node-er-entrance', 'CORRIDOR', 335, true),
    ('I-1301/node-er-entrance>I-1301/node-stairs', 'I-1301/node-er-entrance', 'I-1301/node-stairs', 'CORRIDOR', 335, true),
    ('I-1301/node-lift>I-1301/node-opd-ns', 'I-1301/node-lift', 'I-1301/node-opd-ns', 'CORRIDOR', 125, true),
    ('I-1301/node-opd-ns>I-1301/node-lift', 'I-1301/node-opd-ns', 'I-1301/node-lift', 'CORRIDOR', 125, true),
    ('I-1301/node-opd-ns>I-1301/node-xray', 'I-1301/node-opd-ns', 'I-1301/node-xray', 'CORRIDOR', 360, true),
    ('I-1301/node-xray>I-1301/node-opd-ns', 'I-1301/node-xray', 'I-1301/node-opd-ns', 'CORRIDOR', 360, true),
    ('I-1301/node-xray>I-1301/node-ct', 'I-1301/node-xray', 'I-1301/node-ct', 'CORRIDOR', 180, true),
    ('I-1301/node-ct>I-1301/node-xray', 'I-1301/node-ct', 'I-1301/node-xray', 'CORRIDOR', 180, true),
    ('I-1301/node-ct>I-1301/node-wound-care', 'I-1301/node-ct', 'I-1301/node-wound-care', 'CORRIDOR', 205, true),
    ('I-1301/node-wound-care>I-1301/node-ct', 'I-1301/node-wound-care', 'I-1301/node-ct', 'CORRIDOR', 205, true),
    ('I-1301/node-stairs>I-1301/node-triage', 'I-1301/node-stairs', 'I-1301/node-triage', 'CORRIDOR', 125, true),
    ('I-1301/node-triage>I-1301/node-stairs', 'I-1301/node-triage', 'I-1301/node-stairs', 'CORRIDOR', 125, true),
    ('I-1301/node-triage>I-1301/node-er-exam4', 'I-1301/node-triage', 'I-1301/node-er-exam4', 'CORRIDOR', 150, true),
    ('I-1301/node-er-exam4>I-1301/node-triage', 'I-1301/node-er-exam4', 'I-1301/node-triage', 'CORRIDOR', 150, true),
    ('I-1301/node-er-exam4>I-1301/node-wound-care', 'I-1301/node-er-exam4', 'I-1301/node-wound-care', 'CORRIDOR', 75, true),
    ('I-1301/node-wound-care>I-1301/node-er-exam4', 'I-1301/node-wound-care', 'I-1301/node-er-exam4', 'CORRIDOR', 75, true),
    ('I-1301/node-lift>I-1302/node-lift', 'I-1301/node-lift', 'I-1302/node-lift', 'ELEVATOR', 60, true),
    ('I-1302/node-lift>I-1301/node-lift', 'I-1302/node-lift', 'I-1301/node-lift', 'ELEVATOR', 60, true),
    ('I-1301/node-stairs>I-1302/node-stairs', 'I-1301/node-stairs', 'I-1302/node-stairs', 'STAIRS', 60, false),
    ('I-1302/node-stairs>I-1301/node-stairs', 'I-1302/node-stairs', 'I-1301/node-stairs', 'STAIRS', 60, false),
    ('I-1302/node-lift>I-1302/node-wellness-ns', 'I-1302/node-lift', 'I-1302/node-wellness-ns', 'CORRIDOR', 105, true),
    ('I-1302/node-wellness-ns>I-1302/node-lift', 'I-1302/node-wellness-ns', 'I-1302/node-lift', 'CORRIDOR', 105, true),
    ('I-1302/node-wellness-ns>I-1302/node-pre-counselling', 'I-1302/node-wellness-ns', 'I-1302/node-pre-counselling', 'CORRIDOR', 143, true),
    ('I-1302/node-pre-counselling>I-1302/node-wellness-ns', 'I-1302/node-pre-counselling', 'I-1302/node-wellness-ns', 'CORRIDOR', 143, true),
    ('I-1302/node-wellness-ns>I-1302/node-bof-report', 'I-1302/node-wellness-ns', 'I-1302/node-bof-report', 'CORRIDOR', 163, true),
    ('I-1302/node-bof-report>I-1302/node-wellness-ns', 'I-1302/node-bof-report', 'I-1302/node-wellness-ns', 'CORRIDOR', 163, true),
    ('I-1302/node-bof-report>I-1302/node-blood-collection', 'I-1302/node-bof-report', 'I-1302/node-blood-collection', 'CORRIDOR', 157, true),
    ('I-1302/node-blood-collection>I-1302/node-bof-report', 'I-1302/node-blood-collection', 'I-1302/node-bof-report', 'CORRIDOR', 157, true),
    ('I-1302/node-lift>I-1302/node-longevity', 'I-1302/node-lift', 'I-1302/node-longevity', 'CORRIDOR', 238, true),
    ('I-1302/node-longevity>I-1302/node-lift', 'I-1302/node-longevity', 'I-1302/node-lift', 'CORRIDOR', 238, true),
    ('I-1302/node-longevity>I-1302/node-stairs', 'I-1302/node-longevity', 'I-1302/node-stairs', 'CORRIDOR', 237, true),
    ('I-1302/node-stairs>I-1302/node-longevity', 'I-1302/node-stairs', 'I-1302/node-longevity', 'CORRIDOR', 237, true),
    ('I-1302/node-lift>I-1301/node-lift', 'I-1302/node-lift', 'I-1301/node-lift', 'ELEVATOR', 60, true),
    ('I-1301/node-lift>I-1302/node-lift', 'I-1301/node-lift', 'I-1302/node-lift', 'ELEVATOR', 60, true),
    ('I-1302/node-stairs>I-1301/node-stairs', 'I-1302/node-stairs', 'I-1301/node-stairs', 'STAIRS', 60, false),
    ('I-1301/node-stairs>I-1302/node-stairs', 'I-1301/node-stairs', 'I-1302/node-stairs', 'STAIRS', 60, false) ON CONFLICT (edge_id) DO NOTHING;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'place_entry_node_id_fkey'
    ) THEN
        ALTER TABLE carepath.place
            ADD CONSTRAINT place_entry_node_id_fkey
            FOREIGN KEY (entry_node_id) REFERENCES carepath.nav_node (node_id);
    END IF;
END $$;
