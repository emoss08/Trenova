--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- What became of a telematics arrival or departure: recorded on its stop, a
-- repeat of one already recorded, or left for a dispatcher because it could
-- not be tied to a stop or the stop refused it. The last two are the source of
-- the watchtower's stop visit items.
ALTER TABLE "telematics_events"
    ADD COLUMN IF NOT EXISTS "stop_outcome" varchar(20),
    ADD COLUMN IF NOT EXISTS "stop_visit" varchar(20),
    ADD COLUMN IF NOT EXISTS "shipment_move_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "stop_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "stop_outcome_reason" text;

--bun:split
ALTER TABLE "telematics_events"
    ADD CONSTRAINT "chk_telematics_events_stop_outcome" CHECK (
        "stop_outcome" IS NULL
        OR (
            "stop_outcome" IN ('Recorded', 'Duplicate', 'Unmatched', 'Refused')
            AND "stop_visit" IN ('Arrival', 'Departure')
            AND (
                "stop_outcome" NOT IN ('Unmatched', 'Refused')
                OR ("shipment_move_id" IS NOT NULL AND btrim(coalesce("stop_outcome_reason", '')) <> '')
            )
        )
    );

--bun:split
CREATE INDEX IF NOT EXISTS idx_telematics_events_stop_review
    ON "telematics_events" ("organization_id", "business_unit_id", "occurred_at" DESC)
    WHERE "stop_outcome" IN ('Unmatched', 'Refused');
