-- A memory written by a run that had read outside content taints every turn
-- that reads it, which holds that turn's writes for a person. A person who has
-- read such a memory and kept it (approving it as a suggestion, or reviewing
-- it in AI Control) makes it the organization's own: it stops tainting turns
-- and is followed as written. tainted keeps where the memory came from; the
-- review is who cleared it and when.
ALTER TABLE "agent_memories"
    ADD COLUMN IF NOT EXISTS "reviewed_by_user_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "reviewed_at" bigint;

COMMENT ON COLUMN "agent_memories"."reviewed_by_user_id" IS 'The person who read the memory and kept it; a reviewed memory no longer taints the turns that read it';

COMMENT ON COLUMN "agent_memories"."reviewed_at" IS 'When a person reviewed the memory, as Unix seconds';

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_review";

--bun:split

ALTER TABLE "agent_memories"
    ADD CONSTRAINT "chk_agent_memories_review" CHECK (
        ("reviewed_at" IS NULL) = ("reviewed_by_user_id" IS NULL)
    );

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_unreviewed_taint"
    ON "agent_memories" ("organization_id", "business_unit_id")
    WHERE "tainted" AND "reviewed_at" IS NULL AND "status" = 'Active';
