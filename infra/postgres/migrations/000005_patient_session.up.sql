CREATE TABLE IF NOT EXISTS carepath.patient_session (
    token        text PRIMARY KEY,
    source       text NOT NULL,
    external_id  text NOT NULL,
    display_name text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS patient_session_expires_at_idx ON carepath.patient_session (expires_at);
