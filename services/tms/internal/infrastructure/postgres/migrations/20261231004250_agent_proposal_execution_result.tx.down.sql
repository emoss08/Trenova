-- Drops what executed proposals made. Decision notes and later turns fall
-- back to saying only that the change ran.
ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "execution_result";
