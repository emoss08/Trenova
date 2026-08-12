-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260306150000_assignment_one_per_move.tx.up.sql

CREATE UNIQUE INDEX IF NOT EXISTS "uq_assignments_move_tenant"
    ON "assignments" ("shipment_move_id", "organization_id", "business_unit_id");
