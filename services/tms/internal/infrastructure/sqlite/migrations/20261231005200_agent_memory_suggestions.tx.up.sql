-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231005200_agent_memory_suggestions.tx.up.sql

ALTER TABLE "agent_memories" ADD COLUMN "evidence" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_suggestions"
    ON "agent_memories" ("organization_id", "business_unit_id", "agent_definition_id", "status")WHERE "source" = 'Feedback';
