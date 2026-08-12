-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260727000000_recurring_shipment.tx.up.sql

CREATE TABLE IF NOT EXISTS "recurring_shipments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "source_shipment_id" TEXT NOT NULL,
    "customer_id" TEXT,
    "origin_location_id" TEXT,
    "destination_location_id" TEXT,
    "entered_by_id" TEXT,
    "last_generated_shipment_id" TEXT,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "cron_expression" TEXT NOT NULL,
    "timezone" TEXT NOT NULL,
    "start_date" INTEGER,
    "end_date" INTEGER,
    "max_occurrences" INTEGER,
    "lead_time_days" INTEGER NOT NULL DEFAULT 1,
    "skip_weekends" INTEGER NOT NULL DEFAULT 0,
    "exception_policy" TEXT NOT NULL DEFAULT 'Skip',
    "blackout_dates" TEXT,
    "auto_generate" INTEGER NOT NULL DEFAULT 1,
    "next_occurrence_at" INTEGER,
    "next_occurrence_source_at" INTEGER,
    "last_occurrence_at" INTEGER,
    "last_run_at" INTEGER,
    "generation_count" INTEGER NOT NULL DEFAULT 0,
    "consecutive_failures" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_recurring_shipments" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_recurring_shipments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipments_source_shipment" FOREIGN KEY ("source_shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_recurring_shipments_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON DELETE SET NULL,
    CONSTRAINT "fk_recurring_shipments_origin_location" FOREIGN KEY ("origin_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON DELETE SET NULL,
    CONSTRAINT "fk_recurring_shipments_destination_location" FOREIGN KEY ("destination_location_id", "business_unit_id", "organization_id") REFERENCES "locations"("id", "business_unit_id", "organization_id") ON DELETE SET NULL,
    CONSTRAINT "fk_recurring_shipments_entered_by" FOREIGN KEY ("entered_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "chk_recurring_shipments_lead_time" CHECK (lead_time_days BETWEEN 0 AND 60),
    CONSTRAINT "chk_recurring_shipments_max_occurrences" CHECK (max_occurrences IS NULL OR max_occurrences >= 1),
    CONSTRAINT "chk_recurring_shipments_end_date" CHECK (end_date IS NULL OR start_date IS NULL OR end_date > start_date)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_bu_org" ON "recurring_shipments" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_due_scan" ON "recurring_shipments" ("next_occurrence_at")WHERE
    status = 'Active' AND auto_generate = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_lane_match" ON "recurring_shipments" ("organization_id", "business_unit_id", "customer_id", "origin_location_id", "destination_location_id")WHERE
    status = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_source_shipment" ON "recurring_shipments" ("source_shipment_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_created_at" ON "recurring_shipments" ("created_at", "id");

--bun:split

CREATE TABLE IF NOT EXISTS "recurring_shipment_runs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "recurring_shipment_id" TEXT NOT NULL,
    "generated_shipment_id" TEXT,
    "triggered_by_id" TEXT,
    "status" TEXT NOT NULL,
    "trigger" TEXT NOT NULL,
    "occurrence_at" INTEGER NOT NULL,
    "original_occurrence_at" INTEGER,
    "detail" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_recurring_shipment_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_recurring_shipment_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipment_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipment_runs_series" FOREIGN KEY ("recurring_shipment_id", "business_unit_id", "organization_id") REFERENCES "recurring_shipments"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipment_runs_generated_shipment" FOREIGN KEY ("generated_shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON DELETE SET NULL,
    CONSTRAINT "fk_recurring_shipment_runs_triggered_by" FOREIGN KEY ("triggered_by_id") REFERENCES "users"("id") ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_shipment_runs_series" ON "recurring_shipment_runs" ("recurring_shipment_id", "organization_id", "business_unit_id", "occurrence_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_recurring_shipment_runs_occurrence" ON "recurring_shipment_runs" ("recurring_shipment_id", "occurrence_at", "status");
