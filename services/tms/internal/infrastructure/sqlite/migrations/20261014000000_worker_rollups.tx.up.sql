-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261014000000_worker_rollups.tx.up.sql

ALTER TABLE "worker_profiles" ADD COLUMN "training_health" TEXT NOT NULL DEFAULT 'Current';

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "safety_rating" TEXT NOT NULL DEFAULT 'Excellent';

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "safety_score" INTEGER NOT NULL DEFAULT 100;

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "next_credential_expiry" INTEGER;

--bun:split

ALTER TABLE "worker_profiles" ADD COLUMN "next_training_due" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_rollups"
    ON "worker_profiles" ("organization_id", "business_unit_id", "compliance_status", "training_health", "safety_rating");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_profiles_next_credential_expiry"
    ON "worker_profiles" ("organization_id", "business_unit_id", "next_credential_expiry")WHERE "next_credential_expiry" IS NOT NULL;
