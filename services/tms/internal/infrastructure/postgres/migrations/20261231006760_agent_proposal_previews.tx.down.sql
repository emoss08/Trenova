ALTER TABLE "agent_decisions"
    DROP CONSTRAINT IF EXISTS "ck_agent_decisions_preview_reviewed";

--bun:split

ALTER TABLE "agent_decisions"
    DROP CONSTRAINT IF EXISTS "ck_agent_decisions_preview_digest";

--bun:split

ALTER TABLE "agent_decisions"
    DROP COLUMN IF EXISTS "preview_target_version",
    DROP COLUMN IF EXISTS "preview_reviewed",
    DROP COLUMN IF EXISTS "preview_digest",
    DROP COLUMN IF EXISTS "preview";

--bun:split

DROP TABLE IF EXISTS "agent_proposal_baselines";
