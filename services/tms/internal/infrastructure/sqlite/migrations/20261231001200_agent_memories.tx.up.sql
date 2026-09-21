-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231001200_agent_memories.tx.up.sql

CREATE TABLE IF NOT EXISTS "agent_memories" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "source" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "subject_type" TEXT,
    "subject_id" TEXT,
    "subject_label" TEXT,
    "tool_name" TEXT,
    "content" TEXT NOT NULL,
    "agent_definition_id" TEXT,
    "source_run_id" TEXT,
    "source_proposal_id" TEXT,
    "created_by_user_id" TEXT,
    "retired_by_user_id" TEXT,
    "retired_at" INTEGER,
    "expires_at" INTEGER,
    "use_count" INTEGER NOT NULL DEFAULT 0,
    "last_used_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_agent_memories" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "chk_agent_memories_kind" CHECK ("kind" IN ('Instruction', 'Fact', 'Correction')),
    CONSTRAINT "chk_agent_memories_source" CHECK ("source" IN ('User', 'Agent', 'Decision')),
    CONSTRAINT "chk_agent_memories_status" CHECK ("status" IN ('Active', 'Retired')),
    CONSTRAINT "chk_agent_memories_subject_type" CHECK ("subject_type" IS NULL OR "subject_type" IN ('Customer', 'Location', 'Worker', 'Carrier')),
    CONSTRAINT "chk_agent_memories_subject" CHECK (("subject_type" IS NULL) = ("subject_id" IS NULL)),
    CONSTRAINT "chk_agent_memories_use_count" CHECK ("use_count" >= 0),
    CONSTRAINT "fk_agent_memories_created_by" FOREIGN KEY ("created_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_memories_retired_by" FOREIGN KEY ("retired_by_user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_agent_memories_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_memories_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_active"
    ON "agent_memories" ("organization_id", "business_unit_id", "status", "subject_type", "subject_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_tool"
    ON "agent_memories" ("organization_id", "business_unit_id", "tool_name")WHERE "tool_name" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_agent_memories_created"
    ON "agent_memories" ("organization_id", "business_unit_id", "created_at" DESC);
