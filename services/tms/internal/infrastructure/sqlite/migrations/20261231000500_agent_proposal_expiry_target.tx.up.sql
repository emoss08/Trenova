-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231000500_agent_proposal_expiry_target.tx.up.sql

ALTER TABLE "agent_proposals" ADD COLUMN "expires_at" INTEGER;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "target_resource" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "target_id" TEXT;

--bun:split

ALTER TABLE "agent_proposals" ADD COLUMN "target_version" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposals_pending_expiry"
    ON "agent_proposals" ("expires_at")WHERE "status" = 'Pending' AND "expires_at" IS NOT NULL;
