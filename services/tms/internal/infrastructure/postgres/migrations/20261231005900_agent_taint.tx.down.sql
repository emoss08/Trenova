DROP INDEX IF EXISTS "idx_agent_memories_agent_scope";

--bun:split

ALTER TABLE "agent_memories"
    DROP CONSTRAINT IF EXISTS "chk_agent_memories_scope";

--bun:split

ALTER TABLE "agent_memories"
    DROP COLUMN IF EXISTS "taint_run_id",
    DROP COLUMN IF EXISTS "tainted",
    DROP COLUMN IF EXISTS "scope";

--bun:split

ALTER TABLE "assistant_threads"
    DROP COLUMN IF EXISTS "tainted_at",
    DROP COLUMN IF EXISTS "taint";

--bun:split

DROP INDEX IF EXISTS "idx_agent_proposals_tainted";

--bun:split

ALTER TABLE "agent_proposals"
    DROP CONSTRAINT IF EXISTS "chk_agent_proposals_egress_class";

--bun:split

ALTER TABLE "agent_proposals"
    DROP COLUMN IF EXISTS "held_by",
    DROP COLUMN IF EXISTS "egress_class",
    DROP COLUMN IF EXISTS "taint",
    DROP COLUMN IF EXISTS "tainted";

--bun:split

ALTER TABLE "agent_runs"
    DROP COLUMN IF EXISTS "tainted_at",
    DROP COLUMN IF EXISTS "taint",
    DROP COLUMN IF EXISTS "tainted";
