--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A schedule with an alert delivers only when its run's row count satisfies the
-- condition, which is what turns a recurring report into an exception alert.
ALTER TABLE "report_schedules"
    ADD COLUMN IF NOT EXISTS "alert" jsonb;

-- Suppression state: set while the condition holds so a daily exception report
-- does not mail the same alert every morning until the underlying issue clears.
ALTER TABLE "report_schedules"
    ADD COLUMN IF NOT EXISTS "alert_firing" boolean NOT NULL DEFAULT FALSE;
