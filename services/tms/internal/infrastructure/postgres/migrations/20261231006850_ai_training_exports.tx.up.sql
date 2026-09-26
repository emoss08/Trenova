-- One anonymized training export, run by a Trenova operator across every
-- organization that has consented. It is platform data, so it carries no
-- tenant key; what it took from each organization is in the records table.
CREATE TABLE IF NOT EXISTS "ai_training_exports" (
    "id" varchar(100) NOT NULL,
    "task" varchar(50) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Queued',
    "format" varchar(64) NOT NULL,
    "captured_from" bigint NOT NULL,
    "captured_to" bigint NOT NULL,
    "max_per_organization" integer NOT NULL,
    "validation_percent" integer NOT NULL,
    "requested_by" varchar(255) NOT NULL,
    "note" text,
    "workflow_id" varchar(255),
    "organizations_considered" integer NOT NULL DEFAULT 0,
    "organizations_included" integer NOT NULL DEFAULT 0,
    "examples_total" integer NOT NULL DEFAULT 0,
    "train_examples" integer NOT NULL DEFAULT 0,
    "validation_examples" integer NOT NULL DEFAULT 0,
    "dropped" jsonb NOT NULL DEFAULT '{}',
    "parts" jsonb NOT NULL DEFAULT '[]',
    "progress" jsonb NOT NULL DEFAULT '[]',
    "manifest_key" varchar(512),
    "manifest_sha256" varchar(64),
    "failure_message" text,
    "started_at" bigint,
    "finished_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_ai_training_exports" PRIMARY KEY ("id"),
    CONSTRAINT "ck_ai_training_exports_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    CONSTRAINT "ck_ai_training_exports_status" CHECK ("status" IN ('Queued', 'Running', 'Completed', 'Canceled', 'Failed')),
    CONSTRAINT "ck_ai_training_exports_window" CHECK ("captured_from" >= 0 AND "captured_from" < "captured_to"),
    CONSTRAINT "ck_ai_training_exports_max_per_organization" CHECK ("max_per_organization" BETWEEN 1 AND 20000),
    CONSTRAINT "ck_ai_training_exports_validation_percent" CHECK ("validation_percent" BETWEEN 0 AND 50),
    CONSTRAINT "ck_ai_training_exports_note_length" CHECK ("note" IS NULL OR char_length("note") <= 2000)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_training_exports_created"
    ON "ai_training_exports" ("created_at" DESC);

--bun:split
-- Only one export runs at a time.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_training_exports_active"
    ON "ai_training_exports" ("task")
    WHERE "status" IN ('Queued', 'Running');

--bun:split
-- Which corrections went into which export, under which grant of consent.
CREATE TABLE IF NOT EXISTS "ai_training_export_records" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "export_id" varchar(100) NOT NULL,
    "correction_id" varchar(100) NOT NULL,
    "example_id" varchar(64) NOT NULL,
    "split" varchar(20) NOT NULL,
    "consent_granted_at" bigint NOT NULL,
    "exported_at" bigint NOT NULL,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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

COMMENT ON TABLE "ai_training_exports" IS 'An anonymized model-training export run by a Trenova operator across the organizations that consented';

COMMENT ON COLUMN "ai_training_exports"."requested_by" IS 'The operator who started the export';

COMMENT ON COLUMN "ai_training_exports"."dropped" IS 'Corrections left out, counted by reason: consent withdrawn, no document text, or an identifier that survived anonymization';

COMMENT ON COLUMN "ai_training_exports"."parts" IS 'The JSONL objects written, one per organization and split, with their size and SHA-256; object keys carry no tenant identifier';

COMMENT ON COLUMN "ai_training_exports"."progress" IS 'One entry per organization processed, keyed by its position in the export rather than its id, so a retried organization replaces its own entry';

COMMENT ON TABLE "ai_training_export_records" IS 'Which AI corrections an organization contributed to a training export, and the consent grant in force when they were read';

COMMENT ON COLUMN "ai_training_export_records"."example_id" IS 'The id the example carries in the export files, so it can be removed from a dataset once the organization withdraws consent';

COMMENT ON COLUMN "ai_training_export_records"."consent_granted_at" IS 'When the consent in force at export time was granted';
