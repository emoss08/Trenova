-- The practical and shortest lookup profile assignments had no reader: the product has no
-- point-to-point mileage lookup.
ALTER TABLE "distance_controls" DROP COLUMN IF EXISTS "distance_calculator_shortest_distance_profile_id";

ALTER TABLE "distance_controls" DROP COLUMN IF EXISTS "distance_calculator_practical_distance_profile_id";
