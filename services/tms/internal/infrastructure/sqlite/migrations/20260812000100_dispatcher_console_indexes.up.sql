-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260812000100_dispatcher_console_indexes.up.sql

CREATE INDEX IF NOT EXISTS "idx_shipment_moves_console_coverage" ON "shipment_moves" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_pto_approved_window" ON "worker_pto" ("organization_id", "business_unit_id", "worker_id", "start_date", "end_date")WHERE
    "status" = 'Approved';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_dispatch_capacity" ON "workers" ("organization_id", "business_unit_id", "fleet_code_id")WHERE
    "status" = 'Active'
    AND "can_be_assigned" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assignments_primary_worker_active" ON "assignments" ("primary_worker_id", "organization_id", "business_unit_id")WHERE
    "archived_at" IS NULL;
