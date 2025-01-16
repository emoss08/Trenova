DROP TYPE IF EXISTS "business_unit_status" CASCADE;

DROP TABLE IF EXISTS "business_units" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_business_units_name" CASCADE;

--================================================
--bun:split
DROP INDEX IF EXISTS "idx_business_units_created_at" CASCADE;

