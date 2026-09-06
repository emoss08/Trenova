-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261017000000_worker_injury.tx.up.sql

CREATE TABLE IF NOT EXISTS "worker_injuries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "case_number" INTEGER NOT NULL,
    "case_year" INTEGER NOT NULL,
    "classification" TEXT NOT NULL DEFAULT 'NotRecordable',
    "illness_type" TEXT NOT NULL DEFAULT 'Injury',
    "treatment" TEXT NOT NULL DEFAULT 'None',
    "status" TEXT NOT NULL DEFAULT 'Open',
    "occurred_at" INTEGER NOT NULL,
    "reported_at" INTEGER,
    "returned_to_work_at" INTEGER,
    "location" TEXT,
    "description" TEXT NOT NULL,
    "body_part" TEXT,
    "harmful_agent" TEXT,
    "days_away" INTEGER NOT NULL DEFAULT 0,
    "days_restricted" INTEGER NOT NULL DEFAULT 0,
    "privacy_case" INTEGER NOT NULL DEFAULT 0,
    "claim_status" TEXT NOT NULL DEFAULT 'NotFiled',
    "claim_number" TEXT,
    "claim_carrier" TEXT,
    "claim_filed_at" INTEGER,
    "claim_closed_at" INTEGER,
    "safety_event_id" TEXT,
    "document_id" TEXT,
    "notes" TEXT,
    "recorded_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_worker_injuries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_worker_injuries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_injuries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_injuries_worker" FOREIGN KEY ("worker_id", "organization_id", "business_unit_id") REFERENCES "workers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_worker_injuries_safety_event" FOREIGN KEY ("safety_event_id", "organization_id", "business_unit_id") REFERENCES "worker_safety_events"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_worker_injuries_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_worker_injuries_days" CHECK ("days_away" >= 0 AND "days_away" <= 180 AND "days_restricted" >= 0 AND "days_restricted" <= 180),
    CONSTRAINT "chk_worker_injuries_case_number" CHECK ("case_number" > 0),
    CONSTRAINT "chk_worker_injuries_claim" CHECK ("claim_status" = 'NotFiled' OR "claim_filed_at" IS NOT NULL),
    CONSTRAINT "chk_worker_injuries_returned" CHECK ("returned_to_work_at" IS NULL OR "returned_to_work_at" >= "occurred_at")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_worker_injuries_case" ON "worker_injuries" ("organization_id", "business_unit_id", "case_year", "case_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_injuries_worker" ON "worker_injuries" ("worker_id", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_injuries_log" ON "worker_injuries" ("organization_id", "business_unit_id", "case_year", "classification");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_worker_injuries_open" ON "worker_injuries" ("organization_id", "business_unit_id", "status")WHERE
    "status" = 'Open';

--bun:split

CREATE TABLE IF NOT EXISTS "osha_annual_summaries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "year" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "naics_code" TEXT,
    "average_employees" INTEGER NOT NULL DEFAULT 0,
    "total_hours_worked" INTEGER NOT NULL DEFAULT 0,
    "executive_name" TEXT,
    "executive_title" TEXT,
    "executive_phone" TEXT,
    "certified_at" INTEGER,
    "certified_by_id" TEXT,
    "posted_from" INTEGER,
    "posted_through" INTEGER,
    "submitted_at" INTEGER,
    "submission_reference" TEXT,
    "notes" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_osha_annual_summaries" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_osha_annual_summaries_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_osha_annual_summaries_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_osha_annual_summaries_year" CHECK ("year" >= 1971 AND "year" <= 2200),
    CONSTRAINT "chk_osha_annual_summaries_counts" CHECK ("average_employees" >= 0 AND "total_hours_worked" >= 0),
    CONSTRAINT "chk_osha_annual_summaries_certified" CHECK (("status" = 'Certified') = ("certified_at" IS NOT NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_osha_annual_summaries_year" ON "osha_annual_summaries" ("organization_id", "business_unit_id", "year");
