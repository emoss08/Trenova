-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260527120000_access_policy_uniqueness.tx.up.sql

CREATE UNIQUE INDEX IF NOT EXISTS "idx_access_policies_name_tenant"
    ON "access_policies" (lower("name"), "organization_id", "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_access_policies_enabled_scope_tenant"
    ON "access_policies" ("organization_id", "business_unit_id", "resource", "operation", "conditions")WHERE "enabled" = TRUE;
