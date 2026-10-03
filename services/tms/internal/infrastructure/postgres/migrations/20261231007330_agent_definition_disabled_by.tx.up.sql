-- Who turned an agent off, and when, so a conversation with it can say so
-- instead of only that it can no longer continue.
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "disabled_at" BIGINT,
    ADD COLUMN IF NOT EXISTS "disabled_by_id" VARCHAR(100);
