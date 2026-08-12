-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260724120000_telematics_forms_mapping.tx.up.sql

ALTER TABLE "dispatch_controls" ADD COLUMN "enable_auto_stop_actuals" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE TABLE IF NOT EXISTS "telematics_form_submissions"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL DEFAULT 'Samsara',
    "provider_submission_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "template_name" TEXT,
    "worker_id" TEXT,
    "shipment_id" TEXT,
    "shipment_move_id" TEXT,
    "stop_id" TEXT,
    "submitted_at" INTEGER NOT NULL,
    "fields" TEXT,
    "applied" INTEGER NOT NULL DEFAULT 0,
    "applied_fields" INTEGER NOT NULL DEFAULT 0,
    "applied_at" INTEGER,
    "created_at" INTEGER NOT NULL,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_form_submissions_provider_id
    ON "telematics_form_submissions" ("organization_id", "business_unit_id", "provider", "provider_submission_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_telematics_form_submissions_shipment
    ON "telematics_form_submissions" ("organization_id", "business_unit_id", "shipment_id", "submitted_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_telematics_form_submissions_submitted_at
    ON "telematics_form_submissions" ("organization_id", "business_unit_id", "submitted_at" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "telematics_form_mappings"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "provider" TEXT NOT NULL DEFAULT 'Samsara',
    "template_id" TEXT NOT NULL,
    "template_name" TEXT,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_form_mappings_provider_template
    ON "telematics_form_mappings" ("organization_id", "business_unit_id", "provider", "template_id");

--bun:split

CREATE TABLE IF NOT EXISTS "telematics_form_mapping_items"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "mapping_id" TEXT NOT NULL,
    "source_field_label" TEXT NOT NULL,
    "target_kind" TEXT NOT NULL,
    "target_field" TEXT,
    "target_custom_field_key" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_telematics_form_mapping_items_mapping
    ON "telematics_form_mapping_items" ("organization_id", "business_unit_id", "mapping_id");
