ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_coverage_risk_window_hours";

--bun:split

ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "coverage_risk_window_hours";
