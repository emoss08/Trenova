CREATE TYPE "fuel_surcharge_percent_basis_enum" AS ENUM('Linehaul', 'LinehaulPlusAccessorials');

--bun:split
ALTER TABLE "fuel_surcharge_programs"
    ADD COLUMN IF NOT EXISTS "percent_basis" fuel_surcharge_percent_basis_enum NOT NULL DEFAULT 'Linehaul';

--bun:split
CREATE TYPE "customer_fuel_surcharge_mode_enum" AS ENUM('None', 'Program', 'FuelIncluded');

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "fuel_surcharge_mode" customer_fuel_surcharge_mode_enum NOT NULL DEFAULT 'None';

--bun:split
UPDATE
    "customer_billing_profiles"
SET
    "fuel_surcharge_mode" = 'Program'
WHERE
    "fuel_surcharge_program_id" IS NOT NULL;
