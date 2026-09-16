-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261227000100_agent_proposal_execution_columns.tx.up.sql

ALTER TABLE "agent_proposals" ADD COLUMN "executed_at" INTEGER;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "execution_error" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "source_message_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_source_message" ON "agent_proposals" ("source_message_id")WHERE "source_message_id" IS NOT NULL;
