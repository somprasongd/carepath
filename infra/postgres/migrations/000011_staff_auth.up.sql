-- Staff/admin authentication (ADR-0010, #43): users with argon2id password
-- hashes, a many-to-many role model, and rotating single-use refresh tokens.
-- The patient session module keeps its own table (patient_session) — the two
-- identity kinds are never merged.

CREATE TABLE IF NOT EXISTS carepath.app_user (
    user_id       text PRIMARY KEY,
    username      text NOT NULL UNIQUE,
    password_hash text NOT NULL,          -- argon2id PHC string; never plaintext (NFR-08)
    full_name     text NOT NULL,
    is_active     boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS carepath.role (
    role_id text PRIMARY KEY,
    code    text NOT NULL UNIQUE,         -- MVP: 'ADMIN', 'STAFF'
    name    text NOT NULL
);

CREATE TABLE IF NOT EXISTS carepath.user_role (
    user_id text NOT NULL REFERENCES carepath.app_user (user_id) ON DELETE CASCADE,
    role_id text NOT NULL REFERENCES carepath.role (role_id),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE IF NOT EXISTS carepath.refresh_token (
    token_hash text PRIMARY KEY,          -- sha256 of the opaque token, never the token
    user_id    text NOT NULL REFERENCES carepath.app_user (user_id) ON DELETE CASCADE,
    issued_at  timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,               -- non-null = spent by a rotation
    revoked_at timestamptz                -- non-null = logged out or revoked by reuse detection
);

CREATE INDEX IF NOT EXISTS refresh_token_user_idx ON carepath.refresh_token (user_id);
CREATE INDEX IF NOT EXISTS refresh_token_expires_at_idx ON carepath.refresh_token (expires_at);

INSERT INTO carepath.role (role_id, code, name) VALUES
    ('role-admin', 'ADMIN', 'Administrator'),
    ('role-staff', 'STAFF', 'Staff')
ON CONFLICT (role_id) DO NOTHING;

-- Demo credentials in a public repository: both passwords are "demo", and the
-- argon2id hashes below are committed, so they are known to everyone. They
-- exist for the hackathon demo and local development only — see ADR-0010 §9
-- for the production checklist. ON CONFLICT DO NOTHING keeps a password
-- changed in a live database from being reset by a re-run of migrations.
INSERT INTO carepath.app_user (user_id, username, password_hash, full_name) VALUES
    ('user-admin', 'admin',
     '$argon2id$v=19$m=65536,t=3,p=2$6cQYsXhfiy53rxDIgSsMNA$YZ4Y+NsB9qU90YiJ3Yu9dZ7/HCCyjXwJ6Q6nLjmxWI0',
     'Demo Administrator'),
    ('user-staff', 'staff',
     '$argon2id$v=19$m=65536,t=3,p=2$edCrKYCnF4fd9sVqET6Aeg$7pBauHGQylj7XN9uRqG9smLSntIe0qr8DA67hiPu3dk',
     'Demo Staff')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO carepath.user_role (user_id, role_id) VALUES
    ('user-admin', 'role-admin'),
    ('user-staff', 'role-staff')
ON CONFLICT (user_id, role_id) DO NOTHING;
