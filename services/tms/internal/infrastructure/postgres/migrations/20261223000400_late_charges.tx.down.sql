DROP TABLE IF EXISTS "late_charge_assessments";

--bun:split
ALTER TABLE "billing_controls"
    DROP COLUMN IF EXISTS "late_charge_assessment_mode",
    DROP COLUMN IF EXISTS "late_charge_minimum_amount";

--bun:split
DROP TYPE IF EXISTS "late_charge_assessment_mode_enum";
