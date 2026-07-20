ALTER TABLE "additional_charges"
    DROP COLUMN IF EXISTS "fuel_surcharge_detail",
    DROP COLUMN IF EXISTS "fuel_surcharge_program_id";

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP CONSTRAINT IF EXISTS "fk_customer_billing_profiles_fuel_surcharge_program";

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "fuel_surcharge_program_id";

--bun:split
DROP TABLE IF EXISTS "fuel_surcharge_table_rows";

--bun:split
DROP TABLE IF EXISTS "fuel_surcharge_programs";

--bun:split
DROP TABLE IF EXISTS "fuel_index_prices";

--bun:split
DROP TABLE IF EXISTS "fuel_indices";

--bun:split
DROP TYPE IF EXISTS "fuel_surcharge_program_status_enum";

--bun:split
DROP TYPE IF EXISTS "fuel_surcharge_fallback_enum";

--bun:split
DROP TYPE IF EXISTS "fuel_surcharge_rate_rounding_enum";

--bun:split
DROP TYPE IF EXISTS "fuel_surcharge_step_rounding_enum";

--bun:split
DROP TYPE IF EXISTS "fuel_surcharge_date_basis_enum";

--bun:split
DROP TYPE IF EXISTS "fuel_surcharge_method_kind_enum";

--bun:split
DROP TYPE IF EXISTS "fuel_index_source_enum";
