ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "coverage_risk_window_hours" SMALLINT NOT NULL DEFAULT 12;

ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_coverage_risk_window_hours";

ALTER TABLE "dispatch_controls"
    ADD CONSTRAINT "chk_dispatch_controls_coverage_risk_window_hours" CHECK (
        "coverage_risk_window_hours" > 0
        AND "coverage_risk_window_hours" <= 168
    );
