-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251228000001_audit_dlq.tx.up.sql

CREATE TABLE IF NOT EXISTS "audit_dlq_entries" (
    "id" TEXT PRIMARY KEY,
    "original_entry_id" TEXT NOT NULL,
    "entry_data" TEXT NOT NULL,
    "failure_time" INTEGER NOT NULL,
    "retry_count" INTEGER NOT NULL DEFAULT 0,
    "last_error" TEXT,
    "next_retry_at" INTEGER,
    "status" TEXT NOT NULL DEFAULT 'pending',
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "chk_audit_dlq_status" CHECK (status IN ('pending', 'retrying', 'failed', 'recovered'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_audit_dlq_status_retry" ON "audit_dlq_entries" ("status", "next_retry_at")WHERE status IN ('pending', 'retrying');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_audit_dlq_organization" ON "audit_dlq_entries" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_audit_dlq_created" ON "audit_dlq_entries" ("created_at");
