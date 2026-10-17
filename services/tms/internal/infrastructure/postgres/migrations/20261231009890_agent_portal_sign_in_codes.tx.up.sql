ALTER TABLE "agent_portal_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_portal_runs_status";

--bun:split
ALTER TABLE "agent_portal_runs"
    ADD CONSTRAINT "ck_agent_portal_runs_status" CHECK ("status" IN ('Running', 'AwaitingCode', 'Succeeded', 'Failed', 'Stopped', 'TimedOut', 'Strayed'));

--bun:split
ALTER TABLE "agent_portal_runs"
    ADD COLUMN IF NOT EXISTS "code_prompt" TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "code_requested_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "code_attempts" INTEGER NOT NULL DEFAULT 0;

--bun:split
ALTER TABLE "agent_portal_runs"
    ADD CONSTRAINT "ck_agent_portal_runs_code_attempts" CHECK ("code_attempts" >= 0);
