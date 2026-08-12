-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260901010000_dispatch_scoring_signals.up.sql

CREATE INDEX IF NOT EXISTS "idx_sf_shipment_move" ON "service_failures" ("shipment_move_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_assignments_worker_history" ON "assignments" ("primary_worker_id", "organization_id", "business_unit_id", "created_at")WHERE
    "primary_worker_id" IS NOT NULL;
