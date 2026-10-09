--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

UPDATE "agent_runs" SET "trigger" = 'Event' WHERE "trigger" = 'Wait';

--bun:split
ALTER TABLE "agent_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_runs_trigger";

--bun:split
ALTER TABLE "agent_runs"
    ADD CONSTRAINT "ck_agent_runs_trigger" CHECK ("trigger" IN ('Manual', 'Chat', 'Scheduled', 'Event', 'Continuous'));

--bun:split
UPDATE "assistant_turns" SET "origin" = 'Person' WHERE "origin" = 'WaitResolved';

--bun:split
ALTER TABLE "assistant_turns"
    DROP CONSTRAINT IF EXISTS "ck_assistant_turns_origin";

--bun:split
ALTER TABLE "assistant_turns"
    ADD CONSTRAINT "ck_assistant_turns_origin" CHECK ("origin" IN ('Person', 'DecisionFollowUp', 'Scheduled', 'Compaction'));

--bun:split
DELETE FROM "assistant_messages" WHERE "kind" = 'WaitNote';

--bun:split
ALTER TABLE "assistant_messages"
    DROP CONSTRAINT IF EXISTS "ck_assistant_messages_kind";

--bun:split
ALTER TABLE "assistant_messages"
    ADD CONSTRAINT "ck_assistant_messages_kind" CHECK ("kind" IN ('Message', 'DecisionNote', 'Delegated', 'Schedule', 'Compaction', 'Handoff', 'HandoffBrief', 'Steer', 'WorldChange'));

--bun:split
DROP TABLE IF EXISTS "agent_waits";
