INSERT INTO carepath.service_point (id, code, name, place_id)
VALUES
    ('SP-REG', 'REGISTRATION', 'Registration', 'REG-01'),
    ('SP-LAB', 'LAB', 'Laboratory', 'LAB-01'),
    ('SP-PHARMACY', 'PHARMACY', 'Pharmacy', 'PHARMACY-01')
ON CONFLICT (id) DO NOTHING;
