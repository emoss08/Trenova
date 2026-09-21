-- Reverses every column the up added. Rolling back loses the record of what
-- each proposal was pinned to and when it would have expired; pending
-- proposals then never expire on their own and execute without the staleness
-- check, which is the behaviour before this migration.
DROP INDEX IF EXISTS "idx_agent_proposals_pending_expiry";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "target_version";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "target_id";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "target_resource";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "expires_at";
