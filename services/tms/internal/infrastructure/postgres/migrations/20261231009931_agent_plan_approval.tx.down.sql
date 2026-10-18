ALTER TABLE "agent_plans"
    DROP COLUMN IF EXISTS "approval_preview_digest",
    DROP COLUMN IF EXISTS "approval_note",
    DROP COLUMN IF EXISTS "approval_reason_code";
