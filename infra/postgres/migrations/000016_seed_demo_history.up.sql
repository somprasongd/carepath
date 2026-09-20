-- #88 demo history: nine backdated COMPLETED visits so the executive
-- dashboard (#87) has believable numbers the moment the stack resets —
-- no half hour of clicking the Mock HIS console before a demo.
--
-- THIS IS DEMO DATA WRITTEN STRAIGHT INTO THE PROJECTION TABLES
-- (journey_visit / journey_step / journey_step_status_event). It is only
-- acceptable because those tables are CarePath-owned projections
-- (ADR-0009) that no HIS event will ever reference (no his_applied_event
-- rows point at VISIT-H-*, so the ingest poller cannot replan over them).
-- NEVER use this pattern to fabricate clinical history for real patients —
-- real rows only ever come from the ingest/transition paths.
--
-- Times are offsets from an anchor of now() - 4 hours, clamped to today's
-- Bangkok midnight (greatest(...)) so every event stays inside the
-- analytics "today" window (#86 resolves midnight in ANALYTICS_TIMEZONE,
-- default Asia/Bangkok — this seed hardcodes the same zone and assumes the
-- demo stack does not override it). Caveat: demonstrating in the small
-- hours clamps the anchor to midnight, which compresses the history into
-- the first minutes of the day and can stamp a few events slightly into
-- the future — harmless for aggregates, visible only in raw timestamps.
--
-- The story the numbers tell: ORDERTYPE:LAB deliberately waits longest
-- (19–34 min across six visits) so bottleneckServicePointId is always
-- SP-ORDERTYPE-LAB; X-Ray (three visits) and every other point stay well
-- below it. Deterministic by hand (#39 intent), never randomized.
--
-- Idempotent: the whole block is a no-op when any VISIT-H-% row already
-- exists, and the visit/step inserts carry ON CONFLICT DO NOTHING on their
-- primary keys. journey_step_status_event has no natural key to conflict
-- on, hence the top-level guard instead of per-row ON CONFLICT.
DO $$
DECLARE
    a        timestamptz := greatest(
                 now() - interval '4 hours',
                 date_trunc('day', now() AT TIME ZONE 'Asia/Bangkok') AT TIME ZONE 'Asia/Bangkok');
    r        record;
    st       record;
    vid      text;
    pref     text;
    t0       timestamptz;
    diag_key text;
    diag_kind text;
    diag_sp  text;
    s_diag   double precision;
    -- Cumulative minute marks from each visit's t0 (start of registration).
    -- Waits come from the spec row; service times are fixed per kind, and
    -- each next step becomes READY 2 minutes after the previous one is
    -- completed (the walk between counters).
    reg_r double precision; reg_s double precision; reg_c double precision;
    dia_r double precision; dia_s double precision; dia_c double precision;
    cln_r double precision; cln_s double precision; cln_c double precision;
    cas_r double precision; cas_s double precision; cas_c double precision;
    pha_r double precision; pha_s double precision; pha_c double precision;
BEGIN
    IF EXISTS (SELECT 1 FROM carepath.journey_visit WHERE visit_id LIKE 'VISIT-H-%') THEN
        RAISE NOTICE 'seed_demo_history: VISIT-H-%% rows already present, skipping';
        RETURN;
    END IF;

    -- n, patient_name, diag, w_reg, w_diag, w_clinic, w_cash, w_pharm
    -- (every name starts with สาธิต so demo rows are recognizable on sight)
    FOR r IN SELECT * FROM (VALUES
        (1, 'สาธิต มีสุข',   'LAB',  3, 19, 12, 3, 5),
        (2, 'สาธิต ใจดี',    'XRAY', 2,  8, 15, 4, 6),
        (3, 'สาธิต รักษ์ดี',  'LAB',  4, 22, 16, 5, 7),
        (4, 'สาธิต สุขภา',   'XRAY', 3, 11, 18, 3, 8),
        (5, 'สาธิต บุญมี',   'LAB',  2, 25, 14, 4, 6),
        (6, 'สาธิต ปลอดภัย', 'XRAY', 4, 14, 13, 6, 5),
        (7, 'สาธิต เข้มแข็ง', 'LAB', 3, 28, 17, 5, 8),
        (8, 'สาธิต อ่อนหวาน', 'LAB', 5, 31, 15, 4, 7),
        (9, 'สาธิต ยิ้มแย้ม', 'LAB', 2, 34, 13, 6, 9)
    ) AS v(n, patient_name, diag, w_reg, w_diag, w_clinic, w_cash, w_wpharm) LOOP
        vid   := 'VISIT-H-' || lpad(r.n::text, 3, '0');
        pref  := 'PT-H-' || lpad(r.n::text, 3, '0');
        t0    := a + ((r.n - 1) * 16) * interval '1 min';

        diag_key  := r.diag || ':1';
        diag_kind := r.diag;
        diag_sp   := CASE WHEN r.diag = 'LAB' THEN 'SP-ORDERTYPE-LAB' ELSE 'SP-ORDERTYPE-XRAY' END;
        s_diag    := CASE WHEN r.diag = 'LAB' THEN 9 ELSE 7 END;

        reg_r := 0;              reg_s := reg_r + r.w_reg;   reg_c := reg_s + 3;        -- serve 3
        dia_r := reg_c + 2;      dia_s := dia_r + r.w_diag;  dia_c := dia_s + s_diag;
        cln_r := dia_c + 2;      cln_s := cln_r + r.w_clinic; cln_c := cln_s + 10;      -- consult 10
        cas_r := cln_c + 2;      cas_s := cas_r + r.w_cash;  cas_c := cas_s + 3;        -- pay 3
        pha_r := cas_c + 2;      pha_s := pha_r + r.w_wpharm; pha_c := pha_s + 4;       -- dispense 4

        INSERT INTO carepath.journey_visit (visit_id, patient_ref, patient_name, status, synced_at)
        VALUES (vid, pref, r.patient_name, 'COMPLETED', t0 + pha_c * interval '1 min')
        ON CONFLICT (visit_id) DO NOTHING;

        -- Every step gets the same uniform READY→STARTED→COMPLETED chain.
        -- (The real planner auto-completes REGISTRATION without a
        -- READY/STARTED pair; the simplification is deliberate — this is
        -- display data, not a replay of planner behavior.)
        FOR st IN SELECT * FROM (VALUES
            ('REGISTRATION', 'REGISTRATION', 'SP-REG',        NULL,  NULL, 1, reg_r, reg_s, reg_c),
            (diag_key,       diag_kind,      diag_sp,         NULL,  NULL, 2, dia_r, dia_s, dia_c),
            ('CLINIC:MED:1', 'CLINIC',       'SP-CLINIC-MED', 'MED', 1,    3, cln_r, cln_s, cln_c),
            ('CASHIER',      'CASHIER',      'SP-CASHIER',    NULL,  NULL, 4, cas_r, cas_s, cas_c),
            ('PHARMACY',     'PHARMACY',     'SP-PHARMACY',   NULL,  NULL, 5, pha_r, pha_s, pha_c)
        ) AS s(step_key, kind, sp, clinic_code, round_no, ord, t_ready, t_start, t_done) LOOP
            INSERT INTO carepath.journey_step
                (visit_id, step_key, sequence, kind, clinic_code, round, order_refs,
                 status, service_point_id)
            VALUES (vid, st.step_key, st.ord, st.kind, st.clinic_code, st.round_no, '{}',
                    'COMPLETED', st.sp)
            ON CONFLICT (visit_id, step_key) DO NOTHING;

            INSERT INTO carepath.journey_step_status_event
                (visit_id, step_key, kind, service_point_id,
                 from_status, to_status, source, actor_user_id, actor_username, occurred_at)
            VALUES
                (vid, st.step_key, st.kind, st.sp,
                 NULL, 'READY', 'planner', NULL, NULL,
                 t0 + st.t_ready * interval '1 min'),
                (vid, st.step_key, st.kind, st.sp,
                 'READY', 'STARTED', 'staff-web', NULL, 'staff',
                 t0 + st.t_start * interval '1 min'),
                (vid, st.step_key, st.kind, st.sp,
                 'STARTED', 'COMPLETED', 'staff-web', NULL, 'staff',
                 t0 + st.t_done * interval '1 min');
        END LOOP;
    END LOOP;
END $$;
