-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006800_ai_corrections.tx.up.sql

CREATE TABLE IF NOT EXISTS "ai_corrections" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "task" TEXT NOT NULL,
    "source_type" TEXT NOT NULL,
    "source_id" TEXT NOT NULL,
    "document_id" TEXT,
    "subject_type" TEXT NOT NULL,
    "subject_id" TEXT NOT NULL,
    "captured_by_id" TEXT NOT NULL,
    "document_kind" TEXT,
    "document_fingerprint" TEXT,
    "extraction_model" TEXT,
    "extraction_provider_id" TEXT,
    "predicted_confidence" REAL NOT NULL DEFAULT 0,
    "predicted" TEXT NOT NULL,
    "confirmed" TEXT NOT NULL,
    "field_results" TEXT NOT NULL,
    "scored_count" INTEGER NOT NULL DEFAULT 0,
    "correct_count" INTEGER NOT NULL DEFAULT 0,
    "corrected_count" INTEGER NOT NULL DEFAULT 0,
    "missed_count" INTEGER NOT NULL DEFAULT 0,
    "unconfirmed_count" INTEGER NOT NULL DEFAULT 0,
    "unscored_count" INTEGER NOT NULL DEFAULT 0,
    "captured_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_ai_corrections" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_ai_corrections_source" UNIQUE ("organization_id", "business_unit_id", "source_type", "source_id"),
    CONSTRAINT "ck_ai_corrections_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    CONSTRAINT "ck_ai_corrections_source_type" CHECK ("source_type" IN ('DocumentShipmentDraft')),
    CONSTRAINT "ck_ai_corrections_subject_type" CHECK ("subject_type" IN ('Shipment')),
    CONSTRAINT "ck_ai_corrections_confidence" CHECK ("predicted_confidence" >= 0 AND "predicted_confidence" <= 1),
    CONSTRAINT "ck_ai_corrections_counts" CHECK ("scored_count" >= 0 AND "correct_count" >= 0 AND "corrected_count" >= 0 AND "missed_count" >= 0 AND "unconfirmed_count" >= 0 AND "unscored_count" >= 0),
    CONSTRAINT "ck_ai_corrections_scored" CHECK ("scored_count" = "correct_count" + "corrected_count" + "missed_count"),
    CONSTRAINT "fk_ai_corrections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_corrections_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_corrections_created"
    ON "ai_corrections" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_corrections_task_kind"
    ON "ai_corrections" ("organization_id", "business_unit_id", "task", "document_kind", "captured_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_corrections_subject"
    ON "ai_corrections" ("organization_id", "business_unit_id", "subject_type", "subject_id");

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "ai_training_consent" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "ai_training_consent_changed_at" INTEGER;

--bun:split

ALTER TABLE "agent_controls" ADD COLUMN "ai_training_consent_changed_by_id" TEXT;

--bun:split

ALTER TABLE "data_retention" ADD COLUMN "ai_correction_retention_period" INTEGER NOT NULL DEFAULT 730;
