-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261016000000_driver_qualification.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_employment_verifications"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "employer_name" TEXT NOT NULL,
    "employer_dot_number" TEXT,
    "employer_mc_number" TEXT,
    "contact_name" TEXT,
    "contact_phone" TEXT,
    "contact_email" TEXT,
    "employed_from" INTEGER,
    "employed_to" INTEGER,
    "was_dot_regulated" INTEGER NOT NULL DEFAULT 1,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "method" TEXT NOT NULL DEFAULT 'Email',
    "requested_at" INTEGER,
    "response_received_at" INTEGER,
    "last_follow_up_at" INTEGER,
    "follow_up_count" INTEGER NOT NULL DEFAULT 0,
    "drug_alcohol_response_received_at" INTEGER,
    "had_accidents" INTEGER NOT NULL DEFAULT 0,
    "accident_count" INTEGER NOT NULL DEFAULT 0,
    "had_drug_alcohol_violations" INTEGER NOT NULL DEFAULT 0,
    "findings" TEXT,
    "notes" TEXT,
    "document_id" TEXT,
    "requested_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_employment_verifications" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_employment_verifications_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_verifications_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_verifications_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_employment_verifications_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_employment_verifications_period" CHECK ("employed_from" IS NULL OR "employed_to" IS NULL OR "employed_to" >= "employed_from"),
    CONSTRAINT "chk_worker_employment_verifications_received" CHECK (("status" <> 'Received') OR "response_received_at" IS NOT NULL),
    CONSTRAINT "chk_worker_employment_verifications_requested" CHECK ("status" IN ('Pending', 'NotApplicable') OR "requested_at" IS NOT NULL),
    CONSTRAINT "chk_worker_employment_verifications_counts" CHECK ("follow_up_count" >= 0 AND "accident_count" >= 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_employment_verifications_worker" ON "worker_employment_verifications" ("worker_id", "employed_to" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_employment_verifications_outstanding" ON "worker_employment_verifications" ("organization_id", "business_unit_id", "status")WHERE
    "status" IN ('Pending', 'Requested');

--bun:split

ALTER TABLE "data_retention" ADD COLUMN "driver_qualification_retention_period" INTEGER NOT NULL DEFAULT 1095;
