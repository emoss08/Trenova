-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231008460_audit_entries_resource_index.up.sql

CREATE INDEX IF NOT EXISTS "idx_audit_entries_resource_time" ON "audit_entries" ("organization_id", "business_unit_id", "resource_id", "timestamp");
