--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "snoozed_until" bigint;

ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "snooze_anchor" varchar(20);

ALTER TABLE "assistant_threads"
    ADD COLUMN IF NOT EXISTS "snooze_stop_id" varchar(100);

ALTER TABLE "assistant_threads"
    DROP CONSTRAINT IF EXISTS "ck_assistant_threads_snooze";

ALTER TABLE "assistant_threads"
    ADD CONSTRAINT "ck_assistant_threads_snooze" CHECK (
        ("snoozed_until" IS NULL AND "snooze_anchor" IS NULL AND "snooze_stop_id" IS NULL)
        OR ("snoozed_until" IS NOT NULL AND "snooze_anchor" IN ('Time', 'Appointment', 'ETA'))
    );

COMMENT ON COLUMN "assistant_threads"."snoozed_until" IS 'When a snoozed case comes back to the person; null while it is not snoozed';

COMMENT ON COLUMN "assistant_threads"."snooze_anchor" IS 'What the snooze follows: a set time, the next appointment on the shipment (snooze_stop_id) or its ETA, read again whenever the case is';

SELECT trenova_rls.reconcile();
