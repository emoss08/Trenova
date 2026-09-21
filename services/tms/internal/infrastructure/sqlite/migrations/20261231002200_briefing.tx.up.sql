-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231002200_briefing.tx.up.sql

CREATE TABLE IF NOT EXISTS "assistant_briefings"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "role_key" TEXT NOT NULL,
    "user_id" TEXT,
    "briefing_date" TEXT NOT NULL,
    "run_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "headline" TEXT,
    "sections" TEXT NOT NULL DEFAULT '[]',
    "facts" TEXT NOT NULL DEFAULT '{}',
    "narrated" INTEGER NOT NULL DEFAULT 0,
    "failure_reason" TEXT,
    "emailed_at" INTEGER,
    "read_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_assistant_briefings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_assistant_briefings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_briefings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_assistant_briefings_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_assistant_briefings_status" CHECK ("status" IN ('Pending', 'Ready', 'Failed')),
    CONSTRAINT "ck_assistant_briefings_role" CHECK ("role_key" IN ('Dispatch', 'Billing', 'Compliance', 'Leadership', 'General'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_briefings_day" ON "assistant_briefings" ("organization_id", "business_unit_id", "role_key", "briefing_date", COALESCE("user_id", ''));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assistant_briefings_recent" ON "assistant_briefings" ("organization_id", "business_unit_id", "briefing_date" DESC, "role_key");

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "briefing_enabled" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "briefing_hour_local" INTEGER NOT NULL DEFAULT 6;
