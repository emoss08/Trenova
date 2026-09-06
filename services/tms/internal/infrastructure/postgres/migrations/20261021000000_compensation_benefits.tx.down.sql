DROP TABLE IF EXISTS "worker_benefit_enrollments";
--bun:split
DROP TABLE IF EXISTS "benefit_plans";
--bun:split
DROP INDEX IF EXISTS "idx_recurring_deductions_priority";
--bun:split
ALTER TABLE "recurring_deductions"
    DROP CONSTRAINT IF EXISTS "chk_recurring_deductions_priority";
--bun:split
ALTER TABLE "recurring_deductions"
    DROP COLUMN IF EXISTS "priority";
--bun:split
ALTER TABLE "recurring_deductions"
    DROP COLUMN IF EXISTS "issuing_agency";
--bun:split
ALTER TABLE "recurring_deductions"
    DROP COLUMN IF EXISTS "case_number";
--bun:split
ALTER TABLE "recurring_deductions"
    DROP COLUMN IF EXISTS "court_order_number";
--bun:split
ALTER TABLE "recurring_deductions"
    DROP COLUMN IF EXISTS "kind";
--bun:split
ALTER TABLE "recurring_earnings"
    DROP COLUMN IF EXISTS "kind";
--bun:split
DROP TYPE IF EXISTS "benefit_coverage_tier_enum";
--bun:split
DROP TYPE IF EXISTS "benefit_enrollment_status_enum";
--bun:split
DROP TYPE IF EXISTS "benefit_plan_type_enum";
