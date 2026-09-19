-- Seed the X-Ray service point so demo orders of serviceCode XRAY (mock HIS
-- console "Add order") resolve to the real place on the I-1301 floor plan
-- instead of projecting unmapped (follow-up to #22).
INSERT INTO carepath.service_point (id, code, name, place_id)
VALUES ('SP-XRAY', 'XRAY', 'X-Ray', 'XRAY-01')
ON CONFLICT (id) DO NOTHING;
