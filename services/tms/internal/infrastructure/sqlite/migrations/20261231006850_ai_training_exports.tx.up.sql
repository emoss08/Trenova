-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006850_ai_training_exports.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_training_exports" (
    "id" TEXT NOT NULL,
    "task" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Queued',
    "format" TEXT NOT NULL,
    "captured_from" INTEGER NOT NULL,
    "captured_to" INTEGER NOT NULL,
    "max_per_organization" INTEGER NOT NULL,
    "validation_percent" INTEGER NOT NULL,
    "requested_by" TEXT NOT NULL,
    "note" TEXT,
    "workflow_id" TEXT,
    "organizations_considered" INTEGER NOT NULL DEFAULT 0,
    "organizations_included" INTEGER NOT NULL DEFAULT 0,
    "examples_total" INTEGER NOT NULL DEFAULT 0,
    "train_examples" INTEGER NOT NULL DEFAULT 0,
    "validation_examples" INTEGER NOT NULL DEFAULT 0,
    "dropped" TEXT NOT NULL DEFAULT '{}',
    "parts" TEXT NOT NULL DEFAULT '[]',
    "progress" TEXT NOT NULL DEFAULT '[]',
    "manifest_key" TEXT,
    "manifest_sha256" TEXT,
    "failure_message" TEXT,
    "started_at" INTEGER,
    "finished_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_training_exports" PRIMARY KEY ("id"),
    CONSTRAINT "ck_ai_training_exports_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    CONSTRAINT "ck_ai_training_exports_status" CHECK ("status" IN ('Queued', 'Running', 'Completed', 'Canceled', 'Failed')),
    CONSTRAINT "ck_ai_training_exports_window" CHECK ("captured_from" >= 0 AND "captured_from" < "captured_to"),
    CONSTRAINT "ck_ai_training_exports_max_per_organization" CHECK ("max_per_organization" BETWEEN 1 AND 20000),
    CONSTRAINT "ck_ai_training_exports_validation_percent" CHECK ("validation_percent" BETWEEN 0 AND 50),
    CONSTRAINT "ck_ai_training_exports_note_length" CHECK ("note" IS NULL OR length("note") <= 2000)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_training_exports_created"
    ON "ai_training_exports" ("created_at" DESC);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_training_exports_active"
    ON "ai_training_exports" ("task")WHERE "status" IN ('Queued', 'Running');

--bun:split

CREATE TABLE IF NOT EXISTS "ai_training_export_records" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "export_id" TEXT NOT NULL,
    "correction_id" TEXT NOT NULL,
    "example_id" TEXT NOT NULL,
    "split" TEXT NOT NULL,
    "consent_granted_at" INTEGER NOT NULL,
    "exported_at" INTEGER NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_training_export_records" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_ai_training_export_records_correction" UNIQUE ("export_id", "organization_id", "business_unit_id", "correction_id"),
    CONSTRAINT "uq_ai_training_export_records_example" UNIQUE ("export_id", "example_id"),
    CONSTRAINT "ck_ai_training_export_records_split" CHECK ("split" IN ('train', 'validation')),
    CONSTRAINT "fk_ai_training_export_records_export" FOREIGN KEY ("export_id") REFERENCES "ai_training_exports"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_training_export_records_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_training_export_records_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_training_export_records_tenant"
    ON "ai_training_export_records" ("organization_id", "business_unit_id", "export_id");
