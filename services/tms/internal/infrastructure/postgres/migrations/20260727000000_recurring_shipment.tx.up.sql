CREATE TYPE "recurring_shipment_status_enum" AS ENUM('Active', 'Paused', 'Expired');

--bun:split
CREATE TYPE "recurring_shipment_exception_policy_enum" AS ENUM('Skip', 'PreviousBusinessDay', 'NextBusinessDay');

--bun:split
CREATE TYPE "recurring_shipment_run_status_enum" AS ENUM('Generated', 'Skipped', 'Failed');

--bun:split
CREATE TYPE "recurring_shipment_run_trigger_enum" AS ENUM('Auto', 'Manual');

--bun:split
CREATE TABLE IF NOT EXISTS "recurring_shipments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "source_shipment_id" varchar(100) NOT NULL,
    "customer_id" varchar(100),
    "origin_location_id" varchar(100),
    "destination_location_id" varchar(100),
    "entered_by_id" varchar(100),
    "last_generated_shipment_id" varchar(100),
    "name" varchar(100) NOT NULL,
    "description" text,
    "status" recurring_shipment_status_enum NOT NULL DEFAULT 'Active',
    "cron_expression" varchar(100) NOT NULL,
    "timezone" varchar(64) NOT NULL,
    "start_date" bigint,
    "end_date" bigint,
    "max_occurrences" integer,
    "lead_time_days" smallint NOT NULL DEFAULT 1,
    "skip_weekends" boolean NOT NULL DEFAULT FALSE,
    "exception_policy" recurring_shipment_exception_policy_enum NOT NULL DEFAULT 'Skip',
    "blackout_dates" text[],
    "auto_generate" boolean NOT NULL DEFAULT TRUE,
    "next_occurrence_at" bigint,
    "next_occurrence_source_at" bigint,
    "last_occurrence_at" bigint,
    "last_run_at" bigint,
    "generation_count" bigint NOT NULL DEFAULT 0,
    "consecutive_failures" integer NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_bu_org" ON "recurring_shipments"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_due_scan" ON "recurring_shipments"("next_occurrence_at")
WHERE
    status = 'Active' AND auto_generate = TRUE;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_lane_match" ON "recurring_shipments"("organization_id", "business_unit_id", "customer_id", "origin_location_id", "destination_location_id")
WHERE
    status = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_source_shipment" ON "recurring_shipments"("source_shipment_id", "organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_recurring_shipments_created_at" ON "recurring_shipments"("created_at", "id");

--bun:split
CREATE TABLE IF NOT EXISTS "recurring_shipment_runs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "recurring_shipment_id" varchar(100) NOT NULL,
    "generated_shipment_id" varchar(100),
    "triggered_by_id" varchar(100),
    "status" recurring_shipment_run_status_enum NOT NULL,
    "trigger" recurring_shipment_run_trigger_enum NOT NULL,
    "occurrence_at" bigint NOT NULL,
    "original_occurrence_at" bigint,
    "detail" text,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_recurring_shipment_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_recurring_shipment_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipment_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipment_runs_series" FOREIGN KEY ("recurring_shipment_id", "business_unit_id", "organization_id") REFERENCES "recurring_shipments"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_recurring_shipment_runs_generated_shipment" FOREIGN KEY ("generated_shipment_id", "business_unit_id", "organization_id") REFERENCES "shipments"("id", "business_unit_id", "organization_id") ON DELETE SET NULL,
    CONSTRAINT "fk_recurring_shipment_runs_triggered_by" FOREIGN KEY ("triggered_by_id") REFERENCES "users"("id") ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_recurring_shipment_runs_series" ON "recurring_shipment_runs"("recurring_shipment_id", "organization_id", "business_unit_id", "occurrence_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_recurring_shipment_runs_occurrence" ON "recurring_shipment_runs"("recurring_shipment_id", "occurrence_at", "status");
