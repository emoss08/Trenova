-- Drops the reminder mark. Pending proposals then get reminded about again
-- on the next sweep after the column returns, which is the worse of the two
-- ways a rollback could go wrong and still only a duplicate notice.
DROP INDEX IF EXISTS "idx_agent_proposals_pending_unreminded";

--bun:split
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "reminded_at";
