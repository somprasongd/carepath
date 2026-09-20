-- More demo clinic departments for the service point vocabulary: ENT,
-- surgery, ophthalmology, dentistry. Same shape as CLINIC:MED in 000010 —
-- a clinic binds by "CLINIC:<code>" and the mock-his console's clinic
-- picker lists exactly these rows (it reads GET /api/v1/service-points).
-- All clinics share the OPD nurse station place: the MVP floor plan has no
-- per-clinic rooms yet, so every clinic routes there. Giving one its own
-- room later = seed a place + nav entry node, then repoint place_id.
INSERT INTO carepath.service_point (id, code, name, place_id)
VALUES
    ('SP-CLINIC-ENT',   'CLINIC:ENT',   'หู คอ จมูก (ENT)',  'OPD-NS-01'),
    ('SP-CLINIC-SURG',  'CLINIC:SURG',  'ศัลยกรรม (SURG)',   'OPD-NS-01'),
    ('SP-CLINIC-OPHTH', 'CLINIC:OPHTH', 'ตา (OPHTH)',         'OPD-NS-01'),
    ('SP-CLINIC-DENT',  'CLINIC:DENT',  'ทันตกรรม (DENT)',    'OPD-NS-01')
ON CONFLICT (id) DO NOTHING;
