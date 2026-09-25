-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006760_agent_proposal_previews.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_proposal_baselines"(
    "proposal_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "tool_name" TEXT NOT NULL,
    "preview" TEXT NOT NULL,
    "target_version" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_proposal_baselines" PRIMARY KEY ("proposal_id", "organization_id", "business_unit_id"),
    CONSTRAINT "ck_agent_proposal_baselines_target_version" CHECK ("target_version" IS NULL OR "target_version" >= 0),
    CONSTRAINT "fk_agent_proposal_baselines_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_proposal_baselines_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_proposal_baselines_created_at" ON "agent_proposal_baselines" ("created_at");

--bun:split

ALTER TABLE "agent_decisions" ADD COLUMN "preview" TEXT;

--bun:split

ALTER TABLE "agent_decisions" ADD COLUMN "preview_digest" TEXT;

--bun:split

ALTER TABLE "agent_decisions" ADD COLUMN "preview_reviewed" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_decisions" ADD COLUMN "preview_target_version" INTEGER;
