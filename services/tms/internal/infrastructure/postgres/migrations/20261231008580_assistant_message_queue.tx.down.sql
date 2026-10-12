--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DELETE FROM "assistant_messages" WHERE "kind" IN ('Steer', 'WorldChange');

--bun:split
ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split
ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule', 'Compaction', 'Handoff', 'HandoffBrief'));

--bun:split
ALTER TABLE "assistant_messages"
    DROP COLUMN IF EXISTS "world_changes";

--bun:split
DROP TABLE IF EXISTS "assistant_queued_messages";
