-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260307011000_optimize_worker_list_indexes.tx.up.sql

CREATE INDEX IF NOT EXISTS idx_workers_org_bu_created_at_desc
    ON "workers" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_worker_profiles_worker_org_bu
    ON "worker_profiles" ("worker_id", "organization_id", "business_unit_id");
