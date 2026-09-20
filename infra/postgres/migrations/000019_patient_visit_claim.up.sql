-- #96 (FR-18 / ADR-0010 leftover): bind a patient identity to the visits it
-- may read. The claim is keyed by the stable identity (source + external_id),
-- not the 24h session token, so re-logging in keeps ownership; the visit_id
-- FK doubles as existence validation (a claim for an unprojected visit cannot
-- exist). VN knowledge is the only way a claim is minted today — recorded
-- openly in the issue design, not hidden here.
CREATE TABLE IF NOT EXISTS carepath.patient_visit_claim (
    source     text NOT NULL,
    external_id text NOT NULL,
    visit_id   text NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    claimed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (source, external_id, visit_id)
);

CREATE INDEX IF NOT EXISTS patient_visit_claim_visit_idx
    ON carepath.patient_visit_claim (visit_id);
