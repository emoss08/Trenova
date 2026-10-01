--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS idx_telematics_events_stop_review;

--bun:split
ALTER TABLE "telematics_events"
    DROP CONSTRAINT IF EXISTS "chk_telematics_events_stop_outcome";

--bun:split
ALTER TABLE "telematics_events"
    DROP COLUMN IF EXISTS "stop_outcome_reason",
    DROP COLUMN IF EXISTS "stop_id",
    DROP COLUMN IF EXISTS "shipment_move_id",
    DROP COLUMN IF EXISTS "stop_visit",
    DROP COLUMN IF EXISTS "stop_outcome";
