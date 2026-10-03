-- An approval from a person's own conversation commits a few seconds after it
-- is made, unless they undo it first. The decision says when it is due to
-- commit, when it did, or when and by whom it was undone; the workflow that
-- commits it is named so an undo of one of a batch finds the rest.
ALTER TABLE "agent_decisions"
    ADD COLUMN IF NOT EXISTS "commits_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "committed_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "undone_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "undone_by_user_id" VARCHAR(100),
    ADD COLUMN IF NOT EXISTS "commit_workflow_id" VARCHAR(200);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_agent_decisions_commit_workflow"
    ON "agent_decisions"("organization_id", "commit_workflow_id")
    WHERE "commit_workflow_id" IS NOT NULL;

--bun:split
-- A plan approved from a conversation waits out the same window as a whole.
ALTER TABLE "agent_plans"
    ADD COLUMN IF NOT EXISTS "commits_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "undone_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "undone_by_user_id" VARCHAR(100);

--bun:split
ALTER TABLE "agent_plans" DROP CONSTRAINT IF EXISTS "chk_agent_plans_status";

--bun:split
ALTER TABLE "agent_plans"
    ADD CONSTRAINT "chk_agent_plans_status"
    CHECK ("status" IN ('Pending', 'Approving', 'Approved', 'Completed', 'Failed', 'Rejected', 'Expired'));
