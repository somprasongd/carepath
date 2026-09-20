-- Visit share link (FR-24 / C2, ADR-0011): a third, visit-scoped read-only
-- credential for relatives. The table stores sha256(token), never the token
-- itself — the same posture as refresh_token (ADR-0010): a database leak must
-- not leak live links. Fast hash on purpose: tokens are 256-bit random, not
-- passwords. Expired/revoked rows are harmless and stay until explicitly
-- cleaned (out of scope, mirroring the refresh_token decision).
CREATE TABLE IF NOT EXISTS carepath.visit_share_link (
    token_hash text PRIMARY KEY,
    visit_id   text NOT NULL REFERENCES carepath.journey_visit (visit_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz
);

CREATE INDEX IF NOT EXISTS visit_share_link_visit_idx
    ON carepath.visit_share_link (visit_id);
