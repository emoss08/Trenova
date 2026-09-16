DROP INDEX IF EXISTS "idx_agent_proposals_source_message";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "source_message_id";
