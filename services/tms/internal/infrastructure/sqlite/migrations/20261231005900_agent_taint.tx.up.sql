-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005900_agent_taint.tx.up.sql

ALTER TABLE "agent_runs" ADD COLUMN "tainted" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "taint" TEXT;

--bun:split

ALTER TABLE "agent_runs" ADD COLUMN "tainted_at" INTEGER;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "tainted" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "taint" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "egress_class" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "held_by" TEXT NOT NULL DEFAULT '[]';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_tainted"
    ON "agent_proposals" ("organization_id", "business_unit_id", "created_at" DESC)WHERE "tainted";

--bun:split

ALTER TABLE "assistant_threads" ADD COLUMN "taint" TEXT;

--bun:split

ALTER TABLE "assistant_threads" ADD COLUMN "tainted_at" INTEGER;

--bun:split

ALTER TABLE "agent_memories" ADD COLUMN "scope" TEXT NOT NULL DEFAULT 'Organization';

--bun:split

ALTER TABLE "agent_memories" ADD COLUMN "tainted" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_memories" ADD COLUMN "taint_run_id" TEXT;

--bun:split

UPDATE "agent_memories"
SET "scope" = 'Agent'
WHERE "source" = 'Feedback'
  AND "agent_definition_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_agent_scope"
    ON "agent_memories" ("organization_id", "business_unit_id", "agent_definition_id")WHERE "scope" = 'Agent';
