-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231003300_assistant_turns.tx.up.sql

CREATE TABLE IF NOT EXISTS "assistant_turns"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "thread_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "run_id" TEXT,
    "workflow_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "error_message" TEXT,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_assistant_turns" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_assistant_turns_status" CHECK ("status" IN ('Pending', 'Running', 'Completed', 'Refused', 'Stopped', 'Failed')),
    CONSTRAINT "fk_assistant_turns_thread" FOREIGN KEY ("thread_id", "business_unit_id", "organization_id") REFERENCES "assistant_threads"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_turns_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_turns_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_turns_active" ON "assistant_turns" ("organization_id", "thread_id")WHERE "status" IN ('Pending', 'Running');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_turns_thread" ON "assistant_turns" ("organization_id", "business_unit_id", "thread_id", "created_at" DESC);
