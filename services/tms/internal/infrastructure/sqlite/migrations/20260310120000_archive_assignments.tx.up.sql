-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260310120000_archive_assignments.tx.up.sql

ALTER TABLE "assignments" ADD COLUMN "archived_at" INTEGER;

--bun:split

DROP INDEX IF EXISTS "uq_assignments_move_tenant";

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assignments_move_tenant_active"
    ON "assignments" ("shipment_move_id", "organization_id", "business_unit_id")WHERE "archived_at" IS NULL;
