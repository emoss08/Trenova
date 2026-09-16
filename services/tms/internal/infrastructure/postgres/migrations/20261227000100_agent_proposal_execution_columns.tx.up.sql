-- Separate from the enum migration because ALTER TYPE ... ADD VALUE cannot run
-- in a transaction and the new values are not usable until that one commits.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "executed_at" bigint;

ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "execution_error" text;

--bun:split
-- Chat proposals are answered from the conversation they came out of, so the
-- message they belong to is recorded on the proposal rather than the other way
-- round: a message can carry several proposals, and a proposal has exactly one
-- origin.
ALTER TABLE "agent_proposals"
    ADD COLUMN IF NOT EXISTS "source_message_id" varchar(100);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_proposals_source_message" ON "agent_proposals"("source_message_id") WHERE "source_message_id" IS NOT NULL;

COMMENT ON COLUMN "agent_proposals"."executed_at" IS 'When the approved tool actually ran; null means approved but not yet executed';

COMMENT ON COLUMN "agent_proposals"."execution_error" IS 'Why execution failed, kept so an approver can see what went wrong without reading logs';

COMMENT ON COLUMN "agent_proposals"."source_message_id" IS 'Assistant message this proposal came out of, for proposals raised during a conversation';
