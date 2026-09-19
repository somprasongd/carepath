-- #39 demo happy path: Registration → OPD → X-ray → return to doctor →
-- Cashier → Pharmacy. The cashier counter place (CASHIER-01) and its entry
-- node (I-1301/node-cashier) exist since #24/#26, but no service_point row
-- bound the planner's CASHIER step kind to them, so the journey projection
-- left the cashier step unmapped and the patient screen showed no
-- destination for it.
INSERT INTO carepath.service_point (id, code, name, place_id)
VALUES ('SP-CASHIER', 'CASHIER', 'Cashier', 'CASHIER-01')
ON CONFLICT (id) DO NOTHING;
