--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "assistant_threads"
    DROP CONSTRAINT IF EXISTS "ck_assistant_threads_snooze";

--bun:split
ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "snooze_stop_id";

--bun:split
ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "snooze_anchor";

--bun:split
ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "snoozed_until";
