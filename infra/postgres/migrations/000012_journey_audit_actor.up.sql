-- NFR-09 via ADR-0010 §11: the command audit gains *who* made a command.
-- The existing source column keeps naming the surface ("staff-web"); these
-- columns name the authenticated user behind it. Nullable by design — the
-- ingest poller and other system-initiated writes have no actor.
ALTER TABLE carepath.journey_command_audit
    ADD COLUMN IF NOT EXISTS actor_user_id text,
    ADD COLUMN IF NOT EXISTS actor_username text;
