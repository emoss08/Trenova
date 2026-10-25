--keel:notx
-- Code lookups compare lower(code), as every other code index does, so the accessorial charge
-- code index is rebuilt on lower("code") where it was on upper("code").
DROP INDEX CONCURRENTLY IF EXISTS "uq_accessorial_charges_code";

--keel:split
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uq_accessorial_charges_code" ON "accessorial_charges"("organization_id", "business_unit_id", lower("code"));

--keel:split
-- Tractor imports match a tractor by its license plate as well as its code.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_tractors_license_plate" ON "tractors"(lower("license_plate_number"), "organization_id");
