ALTER TABLE carepath.journey_command_audit
    DROP COLUMN IF EXISTS actor_username,
    DROP COLUMN IF EXISTS actor_user_id;
