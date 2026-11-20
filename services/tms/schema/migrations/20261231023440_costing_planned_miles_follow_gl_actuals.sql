-- Planned monthly miles divide fixed costs only while GL actuals are on; the save checks them
-- then and leaves them alone otherwise, and the table holds the same rule.
ALTER TABLE "costing_controls"
    DROP CONSTRAINT IF EXISTS "chk_costing_controls_planned_monthly_miles";

ALTER TABLE "costing_controls"
    ADD CONSTRAINT "chk_costing_controls_planned_monthly_miles" CHECK (
        NOT "gl_actuals_enabled" OR "planned_monthly_miles" IS NULL OR "planned_monthly_miles" > 0
    );
