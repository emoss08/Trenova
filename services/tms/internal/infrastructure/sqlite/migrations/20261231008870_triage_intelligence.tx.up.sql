-- Hand-written: SQLite cannot alter a CHECK constraint, so ai_corrections is
-- rebuilt with the wider task, source and subject checks and its rows copied
-- across. Nothing references the table, so dropping the old one cascades
-- nowhere.
-- Source: 20261231008870_triage_intelligence.tx.up.sql

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_sender_received" ON "inbound_messages" ("organization_id", "business_unit_id", "sender_domain", "received_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_inbound_messages_received" ON "inbound_messages" ("organization_id", "business_unit_id", "received_at" DESC);

--bun:split

CREATE TABLE "ai_corrections_rebuilt" (
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
    CONSTRAINT "ck_ai_corrections_task" CHECK ("task" IN ('ShipmentDraftExtraction', 'InboundClassification')),
    CONSTRAINT "ck_ai_corrections_source_type" CHECK ("source_type" IN ('DocumentShipmentDraft', 'InboundMessage')),
    CONSTRAINT "ck_ai_corrections_subject_type" CHECK ("subject_type" IN ('Shipment', 'InboundMessage')),
    CONSTRAINT "ck_ai_corrections_confidence" CHECK ("predicted_confidence" >= 0 AND "predicted_confidence" <= 1),
    CONSTRAINT "ck_ai_corrections_counts" CHECK ("scored_count" >= 0 AND "correct_count" >= 0 AND "corrected_count" >= 0 AND "missed_count" >= 0 AND "unconfirmed_count" >= 0 AND "unscored_count" >= 0),
    CONSTRAINT "ck_ai_corrections_scored" CHECK ("scored_count" = "correct_count" + "corrected_count" + "missed_count"),
    CONSTRAINT "fk_ai_corrections_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_ai_corrections_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

INSERT INTO "ai_corrections_rebuilt" ("id", "business_unit_id", "organization_id", "task", "source_type", "source_id", "document_id", "subject_type", "subject_id", "captured_by_id", "document_kind", "document_fingerprint", "extraction_model", "extraction_provider_id", "predicted_confidence", "predicted", "confirmed", "field_results", "scored_count", "correct_count", "corrected_count", "missed_count", "unconfirmed_count", "unscored_count", "captured_at", "version", "created_at", "updated_at")
SELECT "id", "business_unit_id", "organization_id", "task", "source_type", "source_id", "document_id", "subject_type", "subject_id", "captured_by_id", "document_kind", "document_fingerprint", "extraction_model", "extraction_provider_id", "predicted_confidence", "predicted", "confirmed", "field_results", "scored_count", "correct_count", "corrected_count", "missed_count", "unconfirmed_count", "unscored_count", "captured_at", "version", "created_at", "updated_at"
FROM "ai_corrections";

--bun:split

DROP TABLE "ai_corrections";

--bun:split

ALTER TABLE "ai_corrections_rebuilt" RENAME TO "ai_corrections";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_corrections_created"
    ON "ai_corrections" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_corrections_task_kind"
    ON "ai_corrections" ("organization_id", "business_unit_id", "task", "document_kind", "captured_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_ai_corrections_subject"
    ON "ai_corrections" ("organization_id", "business_unit_id", "subject_type", "subject_id");
