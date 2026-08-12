-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260901000000_worker_hos_logs.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_hos_logs"(
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "log_start_at" INTEGER NOT NULL,
    "duty_status" TEXT NOT NULL,
    "log_end_at" INTEGER,
    "remark" TEXT,
    "provider" TEXT NOT NULL DEFAULT 'Samsara',
    "provider_vehicle_id" TEXT,
    "received_at" INTEGER NOT NULL,
    PRIMARY KEY ("organization_id", "business_unit_id", "worker_id", "log_start_at")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_worker_hos_logs_start_at
    ON "worker_hos_logs" ("log_start_at");
