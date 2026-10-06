UPDATE "agent_plans" SET "status" = 'Pending' WHERE "status" = 'Approving';

--bun:split
ALTER TABLE "agent_plans" DROP CONSTRAINT IF EXISTS "chk_agent_plans_status";

--bun:split
ALTER TABLE "agent_plans"
    ADD CONSTRAINT "chk_agent_plans_status"
    CHECK ("status" IN ('Pending', 'Approved', 'Completed', 'Failed', 'Rejected', 'Expired'));

--bun:split
ALTER TABLE "agent_plans"
    DROP COLUMN IF EXISTS "commits_at",
    DROP COLUMN IF EXISTS "undone_at",
    DROP COLUMN IF EXISTS "undone_by_user_id";

--bun:split
DROP INDEX IF EXISTS "idx_agent_decisions_commit_workflow";

--bun:split
ALTER TABLE "agent_decisions"
    DROP COLUMN IF EXISTS "commits_at",
    DROP COLUMN IF EXISTS "committed_at",
    DROP COLUMN IF EXISTS "undone_at",
    DROP COLUMN IF EXISTS "undone_by_user_id",
    DROP COLUMN IF EXISTS "commit_workflow_id";
