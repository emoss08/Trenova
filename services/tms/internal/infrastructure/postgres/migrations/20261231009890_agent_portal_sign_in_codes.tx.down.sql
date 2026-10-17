UPDATE "agent_portal_runs"
SET "status" = 'Failed',
    "failure" = 'The portal task was waiting on a sign-in code when sign-in codes were removed.',
    "finished_at" = extract(epoch from current_timestamp)::bigint
WHERE "status" = 'AwaitingCode';

--bun:split
ALTER TABLE "agent_portal_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_portal_runs_code_attempts";

--bun:split
ALTER TABLE "agent_portal_runs"
    DROP COLUMN IF EXISTS "code_attempts",
    DROP COLUMN IF EXISTS "code_requested_at",
    DROP COLUMN IF EXISTS "code_prompt";

--bun:split
ALTER TABLE "agent_portal_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_portal_runs_status";

--bun:split
ALTER TABLE "agent_portal_runs"
    ADD CONSTRAINT "ck_agent_portal_runs_status" CHECK ("status" IN ('Running', 'Succeeded', 'Failed', 'Stopped', 'TimedOut', 'Strayed'));
