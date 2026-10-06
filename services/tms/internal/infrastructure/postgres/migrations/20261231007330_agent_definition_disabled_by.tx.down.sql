ALTER TABLE "agent_definitions"
    DROP COLUMN IF EXISTS "disabled_at",
    DROP COLUMN IF EXISTS "disabled_by_id";
