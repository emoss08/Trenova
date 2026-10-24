-- The approval a plan in its undo window carries until it commits: the reason,
-- the note and the preview digest the person approved. A commit made without
-- its workflow re-settles the plan against them, as the workflow would.
ALTER TABLE "agent_plans"
    ADD COLUMN IF NOT EXISTS "approval_reason_code" VARCHAR(100),
    ADD COLUMN IF NOT EXISTS "approval_note" TEXT,
    ADD COLUMN IF NOT EXISTS "approval_preview_digest" VARCHAR(64);
