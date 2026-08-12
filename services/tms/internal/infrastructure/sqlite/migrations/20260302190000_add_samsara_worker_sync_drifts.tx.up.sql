-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260302190000_add_samsara_worker_sync_drifts.tx.up.sql

CREATE TABLE IF NOT EXISTS "samsara_worker_sync_drifts"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "drift_type" TEXT NOT NULL,
    "worker_name" TEXT NOT NULL,
    "message" TEXT NOT NULL,
    "local_external_id" TEXT,
    "remote_driver_id" TEXT,
    "detected_at" INTEGER NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id", "drift_type")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_worker_sync_drifts_tenant_detected_at
    ON "samsara_worker_sync_drifts" ("organization_id", "business_unit_id", "detected_at" DESC);
