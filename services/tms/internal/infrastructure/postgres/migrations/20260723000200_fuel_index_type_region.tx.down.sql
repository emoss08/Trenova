DROP INDEX IF EXISTS "idx_fuel_indices_region";

--bun:split
ALTER TABLE "fuel_indices"
    DROP COLUMN IF EXISTS "region",
    DROP COLUMN IF EXISTS "fuel_type";

--bun:split
DROP TYPE IF EXISTS "fuel_type_enum";
