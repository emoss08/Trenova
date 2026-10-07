-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231008470_agent_test_prompts.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_test_prompts"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "agent_definition_id" TEXT NOT NULL,
    "prompt" TEXT NOT NULL,
    "created_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_test_prompts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_agent_test_prompts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_test_prompts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_test_prompts_definition" FOREIGN KEY ("agent_definition_id", "business_unit_id", "organization_id") REFERENCES "agent_definitions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_test_prompts_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_test_prompts_prompt" CHECK (length("prompt") BETWEEN 1 AND 2000)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_test_prompts_agent" ON "agent_test_prompts" ("organization_id", "business_unit_id", "agent_definition_id", "created_at");
