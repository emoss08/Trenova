-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231000800_agent_tool_trust.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_tool_trust" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "agent_definition_id" TEXT NOT NULL,
    "tool_name" TEXT NOT NULL,
    "streak" INTEGER NOT NULL DEFAULT 0,
    "approvals" INTEGER NOT NULL DEFAULT 0,
    "modifications" INTEGER NOT NULL DEFAULT 0,
    "rejections" INTEGER NOT NULL DEFAULT 0,
    "execution_failures" INTEGER NOT NULL DEFAULT 0,
    "earned_tier" TEXT,
    "last_decision_at" INTEGER,
    "promoted_at" INTEGER,
    "demoted_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_tool_trust" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_tool_trust_counts" CHECK ("streak" >= 0 AND "approvals" >= 0 AND "modifications" >= 0
        AND "rejections" >= 0 AND "execution_failures" >= 0),
    CONSTRAINT "fk_agent_tool_trust_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_tool_trust_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_tool_trust_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_agent_tool_trust_agent_tool"
    ON "agent_tool_trust" ("organization_id", "business_unit_id", "agent_definition_id", "tool_name");

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "earned_autonomy" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "promotion_threshold" INTEGER NOT NULL DEFAULT 10;
