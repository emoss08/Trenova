-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211015752_audit_entry.tx.up.sql

CREATE TABLE IF NOT EXISTS "audit_entries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "resource" TEXT NOT NULL,
    "resource_id" TEXT NOT NULL,
    "operation" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "timestamp" INTEGER NOT NULL DEFAULT (unixepoch()),
    "changes" TEXT,
    "previous_state" TEXT,
    "current_state" TEXT,
    "metadata" TEXT,
    "user_agent" TEXT,
    "correlation_id" TEXT,
    "comment" TEXT,
    "sensitive_data" INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_audit_entries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_entries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_audit_entries_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_audit_entries_timestamp" ON "audit_entries" ("timestamp");

--bun:split

ALTER TABLE "audit_entries" ADD COLUMN "category" TEXT NOT NULL DEFAULT 'System';

--bun:split

ALTER TABLE "audit_entries" ADD COLUMN "critical" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "audit_entries" ADD COLUMN "ip_address" TEXT;
