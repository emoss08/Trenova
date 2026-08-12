-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260403133000_add_document_ai_extractions.tx.up.sql

CREATE TABLE IF NOT EXISTS "document_ai_extractions" (
    "id" TEXT NOT NULL,
    "document_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "extracted_at" INTEGER NOT NULL,
    "request_hash" TEXT NOT NULL,
    "workflow_id" TEXT NOT NULL,
    "workflow_run_id" TEXT NOT NULL,
    "activity_id" TEXT NOT NULL,
    "task_token" BLOB NOT NULL,
    "response_id" TEXT,
    "model" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "failure_code" TEXT,
    "failure_message" TEXT,
    "submitted_at" INTEGER,
    "last_polled_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_ai_extractions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_ai_extractions_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id")
        REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_ai_extractions_document_extracted_at"
    ON "document_ai_extractions" ("document_id", "organization_id", "business_unit_id", "extracted_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_ai_extractions_status_poll"
    ON "document_ai_extractions" ("status", "last_polled_at", "submitted_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_ai_extractions_response_id"
    ON "document_ai_extractions" ("response_id");
