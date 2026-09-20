-- #136 (FR-18 layer 2): the slip-held visit link. The token a patient's
-- QR/URL carries is minted by CarePath for the HIS to print; only its
-- sha256 is stored (raw never recoverable — ADR-0011 precedent). The token
-- is deterministic HMAC(secret, visit_id:version), so a repeat mint without
-- rotate returns the same URL without ever storing the token itself; a
-- rotate bumps the version and the previous token's hash disappears.
--
-- completed_at on journey_visit records when the projection first saw the
-- visit COMPLETED — the completed-grace policy (VISIT_LINK_COMPLETED_GRACE)
-- is evaluated lazily at claim time against this column, so the event path
-- stays untouched and a grace change applies without data backfill.
ALTER TABLE carepath.journey_visit
    ADD COLUMN IF NOT EXISTS completed_at timestamptz;

CREATE TABLE IF NOT EXISTS carepath.visit_access_token (
    visit_id     text PRIMARY KEY REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    version      integer NOT NULL,
    token_hash   text NOT NULL UNIQUE,
    generated_at timestamptz NOT NULL DEFAULT now()
);
