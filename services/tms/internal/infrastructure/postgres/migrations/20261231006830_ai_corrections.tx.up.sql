-- What an AI extraction predicted beside what a person confirmed when they
-- turned it into a record: a shipment draft read from a rate confirmation or a
-- bill of lading against the shipment created from it. One row per source; a
-- source attached again replaces its row. These rows are the organization's own
-- extraction accuracy and evaluation set. They leave the tenant for model
-- training only while agent_controls.ai_training_consent is on, and only after
-- anonymization at export.
CREATE TABLE IF NOT EXISTS "ai_corrections" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "task" varchar(50) NOT NULL,
    "source_type" varchar(50) NOT NULL,
    "source_id" varchar(100) NOT NULL,
    "document_id" varchar(100),
    "subject_type" varchar(50) NOT NULL,
    "subject_id" varchar(100) NOT NULL,
    "captured_by_id" varchar(100) NOT NULL,
    "document_kind" varchar(100),
    "document_fingerprint" varchar(255),
    "extraction_model" varchar(255),
    "extraction_provider_id" varchar(100),
    "predicted_confidence" double precision NOT NULL DEFAULT 0,
    "predicted" jsonb NOT NULL,
    "confirmed" jsonb NOT NULL,
    "field_results" jsonb NOT NULL,
    "scored_count" integer NOT NULL DEFAULT 0,
    "correct_count" integer NOT NULL DEFAULT 0,
    "corrected_count" integer NOT NULL DEFAULT 0,
    "missed_count" integer NOT NULL DEFAULT 0,
    "unconfirmed_count" integer NOT NULL DEFAULT 0,
    "unscored_count" integer NOT NULL DEFAULT 0,
    "captured_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
-- Retention deletes by time; evaluation reads recent rows.
CREATE INDEX IF NOT EXISTS "idx_ai_corrections_created"
    ON "ai_corrections" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
-- Accuracy is measured per task and per kind of document.
CREATE INDEX IF NOT EXISTS "idx_ai_corrections_task_kind"
    ON "ai_corrections" ("organization_id", "business_unit_id", "task", "document_kind", "captured_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_ai_corrections_subject"
    ON "ai_corrections" ("organization_id", "business_unit_id", "subject_type", "subject_id");

COMMENT ON TABLE "ai_corrections" IS 'An AI extraction''s prediction beside what a person confirmed, field by field, for measuring and improving extraction';

COMMENT ON COLUMN "ai_corrections"."task" IS 'The extraction measured: ShipmentDraftExtraction';

COMMENT ON COLUMN "ai_corrections"."source_type" IS 'What held the prediction: DocumentShipmentDraft';

COMMENT ON COLUMN "ai_corrections"."source_id" IS 'The id of the record that held the prediction, such as the document shipment draft';

COMMENT ON COLUMN "ai_corrections"."document_id" IS 'The document the prediction was read from, when there is one';

COMMENT ON COLUMN "ai_corrections"."subject_type" IS 'What the person confirmed the prediction into: Shipment';

COMMENT ON COLUMN "ai_corrections"."subject_id" IS 'The id of the confirmed record, such as the shipment created from the draft';

COMMENT ON COLUMN "ai_corrections"."captured_by_id" IS 'The person whose action confirmed the record';

COMMENT ON COLUMN "ai_corrections"."document_kind" IS 'The classified kind of the source document, such as RateConfirmation or BillOfLading';

COMMENT ON COLUMN "ai_corrections"."document_fingerprint" IS 'The issuer layout the document was recognised as, used to measure accuracy per layout';

COMMENT ON COLUMN "ai_corrections"."extraction_model" IS 'The model that ran the AI extraction, empty when the prediction came from rules alone';

COMMENT ON COLUMN "ai_corrections"."extraction_provider_id" IS 'The AI provider that served the extraction model';

COMMENT ON COLUMN "ai_corrections"."predicted_confidence" IS 'The draft''s overall confidence, between 0 and 1';

COMMENT ON COLUMN "ai_corrections"."predicted" IS 'The predicted field values and stops, as the draft held them';

COMMENT ON COLUMN "ai_corrections"."confirmed" IS 'The confirmed field values and stops, as the record held them when it was confirmed';

COMMENT ON COLUMN "ai_corrections"."field_results" IS 'Per field: the predicted and confirmed values and the outcome (Correct, Corrected, Missed, Unconfirmed or Unscored)';

COMMENT ON COLUMN "ai_corrections"."scored_count" IS 'Fields with a confirmed value to score against: correct plus corrected plus missed';

COMMENT ON COLUMN "ai_corrections"."unconfirmed_count" IS 'Fields predicted but left empty on the record, so neither right nor wrong';

COMMENT ON COLUMN "ai_corrections"."unscored_count" IS 'Fields present on both sides that could not be compared, such as a date that could not be read';

COMMENT ON COLUMN "ai_corrections"."captured_at" IS 'When the record was confirmed and the correction captured';

--bun:split
-- Whether this organization lets its corrections, anonymized, train models
-- beyond its own tenant. Off by default: it is something an organization
-- opts into, and the change is recorded with who made it and when.
ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "ai_training_consent" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "ai_training_consent_changed_at" bigint,
    ADD COLUMN IF NOT EXISTS "ai_training_consent_changed_by_id" varchar(100);

COMMENT ON COLUMN "agent_controls"."ai_training_consent" IS 'Whether the organization''s AI corrections may be anonymized and used to train models outside the tenant; off by default';

COMMENT ON COLUMN "agent_controls"."ai_training_consent_changed_at" IS 'When training consent was last turned on or off';

COMMENT ON COLUMN "agent_controls"."ai_training_consent_changed_by_id" IS 'Who last turned training consent on or off';

--bun:split
ALTER TABLE "data_retention"
    ADD COLUMN IF NOT EXISTS "ai_correction_retention_period" integer NOT NULL DEFAULT 730;

COMMENT ON COLUMN "data_retention"."ai_correction_retention_period" IS 'Days an AI correction, with the predicted and confirmed values it holds, is kept before it is deleted. At least 30; 730 by default, and zero reads as the default';

--bun:split
ALTER TABLE "data_retention"
    DROP CONSTRAINT IF EXISTS "ck_data_retention_ai_correction";

--bun:split
ALTER TABLE "data_retention"
    ADD CONSTRAINT "ck_data_retention_ai_correction" CHECK ("ai_correction_retention_period" = 0 OR "ai_correction_retention_period" >= 30);
