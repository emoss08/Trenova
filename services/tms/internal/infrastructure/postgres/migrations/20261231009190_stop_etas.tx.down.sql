--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "customers"
    DROP CONSTRAINT IF EXISTS "chk_customers_eta_risk_threshold_minutes";

--bun:split
ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "eta_risk_threshold_minutes";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_eta_risk_threshold_minutes",
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_eta_dwell_minutes";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "eta_dwell_minutes",
    DROP COLUMN IF EXISTS "eta_risk_threshold_minutes",
    DROP COLUMN IF EXISTS "enable_eta_alerts";

--bun:split
DROP TABLE IF EXISTS "stop_etas";
