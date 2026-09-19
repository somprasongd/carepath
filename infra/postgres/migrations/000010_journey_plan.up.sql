-- ADR-0009: CarePath derives the journey plan from HIS facts instead of
-- projecting an HIS-reported step list. journey_step gains a stable
-- step_key identity (sequence becomes display order only), the vocabulary
-- to describe a plan step (kind/clinic_code/round/order_refs), and a
-- journey_closed_round table for the return-to-doctor override (§4).
-- journey_visit gains patient_name (§8 — the one piece of PHI CarePath
-- stores). journey_command_audit addresses by step_key instead of sequence.
--
-- The table is dropped and recreated rather than ALTERed in place: there is
-- no production data yet (ADR-0008/0009 predate any real deployment), and
-- the shape change (new primary key) is easier to read as a clean recreate.

DROP TABLE IF EXISTS carepath.journey_step;

CREATE TABLE carepath.journey_step (
    visit_id         TEXT NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    step_key         TEXT NOT NULL,
    sequence         INT  NOT NULL,
    kind             TEXT NOT NULL,
    clinic_code      TEXT,
    round            INT,
    order_refs       TEXT[] NOT NULL DEFAULT '{}',
    status           TEXT NOT NULL,
    service_point_id TEXT,
    PRIMARY KEY (visit_id, step_key)
);

-- The return-to-doctor override (ADR-0009 §4): a clinic round confirmed
-- finished, whether by the HIS's encounter.completed fact or staff directly.
-- Rows are never deleted for a visit still being planned; a fresh visit_id
-- never collides.
CREATE TABLE carepath.journey_closed_round (
    visit_id  TEXT NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    step_key  TEXT NOT NULL,
    closed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (visit_id, step_key)
);

ALTER TABLE carepath.journey_visit ADD COLUMN IF NOT EXISTS patient_name TEXT;

ALTER TABLE carepath.journey_command_audit DROP COLUMN IF EXISTS sequence;
ALTER TABLE carepath.journey_command_audit ADD COLUMN IF NOT EXISTS step_key TEXT NOT NULL DEFAULT '';
ALTER TABLE carepath.journey_command_audit ALTER COLUMN step_key DROP DEFAULT;

-- ADR-0009 service point vocabulary: a clinic binds by "CLINIC:<code>", a
-- diagnostic step by "ORDERTYPE:<type>"; REGISTRATION/CASHIER/PHARMACY are
-- unchanged fixed codes. Added as new rows alongside LAB/XRAY/DOCTOR rather
-- than renaming them in place: those bare codes are still exercised by
-- other modules' seed-integration tests (navigation, servicepoint) that
-- predate this migration and are not journey-domain concerns to touch here.
-- Multiple service points may point at the same place — no uniqueness
-- constraint on place_id. The demo clinic is MED, reusing the OPD nurse
-- station place seeded in #24.
INSERT INTO carepath.service_point (id, code, name, place_id)
VALUES
    ('SP-ORDERTYPE-LAB',  'ORDERTYPE:LAB',  'Laboratory', 'LAB-01'),
    ('SP-ORDERTYPE-XRAY', 'ORDERTYPE:XRAY', 'X-Ray',      'XRAY-01'),
    ('SP-CLINIC-MED',     'CLINIC:MED',     'อายุรกรรม (MED)', 'OPD-NS-01')
ON CONFLICT (id) DO NOTHING;
