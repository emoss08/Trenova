-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260306133000_add_audit_principal_fields.tx.up.sql

ALTER TABLE "audit_entries" ADD COLUMN "principal_type" TEXT NOT NULL DEFAULT 'session_user';

--bun:split

ALTER TABLE "audit_entries" ADD COLUMN "principal_id" TEXT;

--bun:split

ALTER TABLE "audit_entries" ADD COLUMN "api_key_id" TEXT;

--bun:split

UPDATE "audit_entries"
SET
    "principal_type" = 'session_user',
    "principal_id" = "user_id"
WHERE "principal_id" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_audit_entries_principal" ON "audit_entries" ("principal_type", "principal_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_audit_entries_api_key" ON "audit_entries" ("api_key_id");
