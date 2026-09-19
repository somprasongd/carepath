DELETE FROM carepath.service_point WHERE id IN ('SP-ORDERTYPE-LAB', 'SP-ORDERTYPE-XRAY', 'SP-CLINIC-MED');

ALTER TABLE carepath.journey_command_audit DROP COLUMN IF EXISTS step_key;
ALTER TABLE carepath.journey_command_audit ADD COLUMN IF NOT EXISTS sequence INT NOT NULL DEFAULT 0;
ALTER TABLE carepath.journey_command_audit ALTER COLUMN sequence DROP DEFAULT;

ALTER TABLE carepath.journey_visit DROP COLUMN IF EXISTS patient_name;

DROP TABLE IF EXISTS carepath.journey_closed_round;

DROP TABLE IF EXISTS carepath.journey_step;

CREATE TABLE carepath.journey_step (
    visit_id         TEXT NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    sequence         INT  NOT NULL,
    service_code     TEXT NOT NULL,
    status           TEXT NOT NULL,
    service_point_id TEXT,
    PRIMARY KEY (visit_id, sequence)
);
