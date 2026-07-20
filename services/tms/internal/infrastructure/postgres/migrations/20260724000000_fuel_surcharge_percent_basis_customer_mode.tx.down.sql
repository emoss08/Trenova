ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "fuel_surcharge_mode";

--bun:split
DROP TYPE IF EXISTS "customer_fuel_surcharge_mode_enum";

--bun:split
ALTER TABLE "fuel_surcharge_programs"
    DROP COLUMN IF EXISTS "percent_basis";

--bun:split
DROP TYPE IF EXISTS "fuel_surcharge_percent_basis_enum";
