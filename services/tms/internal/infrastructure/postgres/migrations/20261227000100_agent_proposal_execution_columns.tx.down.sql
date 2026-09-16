-- Reverses every column the up added. Rolling back discards the record of which
-- approved proposals actually ran, which is unavoidable: the enum values that
-- carried the same information cannot be removed from Postgres either, so a
-- rollback leaves proposals whose status says Executed with nothing to back it up.
DROP INDEX IF EXISTS "idx_agent_proposals_source_message";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "source_message_id";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "execution_error";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "executed_at";
