-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261020000000_org_structure.tx.up.sql

CREATE TABLE IF NOT EXISTS "job_positions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "department" TEXT NOT NULL DEFAULT 'Operations',
    "flsa_exempt" INTEGER NOT NULL DEFAULT 0,
    "is_driving_position" INTEGER NOT NULL DEFAULT 1,
    "reports_to_position_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_job_positions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_job_positions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_job_positions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_job_positions_reports_to" FOREIGN KEY ("reports_to_position_id", "organization_id", "business_unit_id") REFERENCES "job_positions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_job_positions_reports_to_self" CHECK ("reports_to_position_id" IS NULL OR "reports_to_position_id" <> "id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_job_positions_code" ON "job_positions" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_job_positions_department" ON "job_positions" ("organization_id", "business_unit_id", "department");

--bun:split

ALTER TABLE "workers" ADD COLUMN "position_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_manager" ON "workers" ("organization_id", "business_unit_id", "manager_id")WHERE
    "manager_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_workers_position" ON "workers" ("organization_id", "business_unit_id", "position_id")WHERE
    "position_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "approval_delegations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "delegator_id" TEXT NOT NULL,
    "delegate_id" TEXT NOT NULL,
    "scope" TEXT NOT NULL DEFAULT 'All',
    "starts_at" INTEGER NOT NULL,
    "ends_at" INTEGER,
    "reason" TEXT,
    "revoked_at" INTEGER,
    "revoked_by_id" TEXT,
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_approval_delegations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_approval_delegations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_approval_delegations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_approval_delegations_delegator" FOREIGN KEY ("delegator_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_approval_delegations_delegate" FOREIGN KEY ("delegate_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_approval_delegations_distinct" CHECK ("delegator_id" <> "delegate_id"),
    CONSTRAINT "chk_approval_delegations_window" CHECK ("ends_at" IS NULL OR "ends_at" >= "starts_at")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_approval_delegations_delegate" ON "approval_delegations" ("organization_id", "business_unit_id", "delegate_id", "starts_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_approval_delegations_delegator" ON "approval_delegations" ("organization_id", "business_unit_id", "delegator_id", "starts_at");
