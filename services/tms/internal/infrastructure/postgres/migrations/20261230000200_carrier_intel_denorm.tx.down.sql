DROP INDEX IF EXISTS "idx_customers_broker_vetting";

--bun:split
ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "broker_vetting_enabled",
    DROP COLUMN IF EXISTS "mc_number",
    DROP COLUMN IF EXISTS "dot_number";

--bun:split
DROP INDEX IF EXISTS "idx_carriers_intel_review_required";

--bun:split
ALTER TABLE "carriers"
    DROP COLUMN IF EXISTS "intel_blocking_count",
    DROP COLUMN IF EXISTS "intel_review_required",
    DROP COLUMN IF EXISTS "intel_risk_level";
