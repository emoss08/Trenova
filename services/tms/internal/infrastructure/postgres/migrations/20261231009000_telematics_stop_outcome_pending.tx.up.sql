--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

-- A telematics visit now becomes a tracking event, and a tracking event that
-- cannot yet reach its stop (an earlier stop is still open) waits rather than
-- being refused. The telematics row records that it is waiting.
ALTER TABLE "telematics_events"
    DROP CONSTRAINT IF EXISTS "chk_telematics_events_stop_outcome";

--bun:split
ALTER TABLE "telematics_events"
    ADD CONSTRAINT "chk_telematics_events_stop_outcome" CHECK (
        "stop_outcome" IS NULL
        OR (
            "stop_outcome" IN ('Recorded', 'Duplicate', 'Unmatched', 'Refused', 'Pending')
            AND "stop_visit" IN ('Arrival', 'Departure')
            AND (
                "stop_outcome" NOT IN ('Unmatched', 'Refused')
                OR ("shipment_move_id" IS NOT NULL AND btrim(coalesce("stop_outcome_reason", '')) <> '')
            )
        )
    );
