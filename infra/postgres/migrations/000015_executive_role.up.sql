-- #86: the EXECUTIVE role and its demo login. Executives see the analytics
-- overview (epic #83) and nothing else: RequireRole lists EXECUTIVE only on
-- the analytics routes, so an executive token is 403 on every other staff
-- surface (ADR-0010 — least privilege per role).

INSERT INTO carepath.role (role_id, code, name) VALUES
    ('role-executive', 'EXECUTIVE', 'Executive')
ON CONFLICT (role_id) DO NOTHING;

-- Demo credential, same caveat as 000011: password "demo", hash committed in
-- a public repo — hackathon/local only (ADR-0010 §9). ON CONFLICT DO NOTHING
-- keeps a password changed in a live database from being reset on re-run.
INSERT INTO carepath.app_user (user_id, username, password_hash, full_name) VALUES
    ('user-exec', 'exec',
     '$argon2id$v=19$m=65536,t=3,p=2$940p63120tpHETCYwDyiEw$pHSe3XYvdpuaSzBy3+wk3gTIzhAOwFGYMbbViFIgDTU',
     'Demo Executive')
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO carepath.user_role (user_id, role_id) VALUES
    ('user-exec', 'role-executive')
ON CONFLICT (user_id, role_id) DO NOTHING;
