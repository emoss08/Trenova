-- The rolling window means something only while GL actuals are on; the save checks it then and
-- leaves it alone otherwise, and the table holds the same rule.
ALTER TABLE "costing_controls"
    DROP CONSTRAINT IF EXISTS "chk_costing_controls_gl_rolling_months";

ALTER TABLE "costing_controls"
    ADD CONSTRAINT "chk_costing_controls_gl_rolling_months" CHECK (
        NOT "gl_actuals_enabled" OR "gl_rolling_months" BETWEEN 1 AND 12
    );
