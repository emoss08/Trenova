-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231000900_agent_proposal_reminder.tx.up.sql

ALTER TABLE "agent_proposals" ADD COLUMN "reminded_at" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_pending_unreminded"
    ON "agent_proposals" ("created_at")WHERE "status" = 'Pending' AND "reminded_at" IS NULL;
