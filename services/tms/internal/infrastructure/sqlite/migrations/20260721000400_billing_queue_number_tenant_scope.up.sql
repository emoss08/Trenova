-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260721000400_billing_queue_number_tenant_scope.up.sql

CREATE UNIQUE INDEX IF NOT EXISTS "uq_billing_queue_items_number_tenant" ON "billing_queue_items" ("number", "organization_id", "business_unit_id");

--bun:split

DROP INDEX IF EXISTS "uq_billing_queue_items_number";
