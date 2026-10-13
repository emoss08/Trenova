--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "customers"
    DROP CONSTRAINT IF EXISTS "chk_customers_check_call_interval_hours";

--bun:split
ALTER TABLE "customers"
    DROP COLUMN IF EXISTS "check_call_interval_hours";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP CONSTRAINT IF EXISTS "chk_dispatch_controls_check_call_interval_hours";

--bun:split
ALTER TABLE "dispatch_controls"
    DROP COLUMN IF EXISTS "check_call_interval_hours",
    DROP COLUMN IF EXISTS "enable_check_call_monitoring";

--bun:split
DELETE FROM "tracking_events" WHERE "source" = 'Carrier';

--bun:split
ALTER TABLE "tracking_events"
    DROP CONSTRAINT IF EXISTS "chk_tracking_events_source";

--bun:split
ALTER TABLE "tracking_events"
    ADD CONSTRAINT "chk_tracking_events_source" CHECK ("source" IN ('Dispatcher', 'Agent', 'Driver', 'Telematics', 'EDI'));

--bun:split
DROP TABLE IF EXISTS "carrier_tracking_links";

--bun:split
DROP TABLE IF EXISTS "check_calls";
