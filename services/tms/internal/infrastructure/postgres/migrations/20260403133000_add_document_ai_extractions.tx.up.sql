CREATE TABLE IF NOT EXISTS "document_ai_extractions" (
    "id" VARCHAR(100) NOT NULL,
    "document_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "user_id" VARCHAR(100) NOT NULL,
    "extracted_at" BIGINT NOT NULL,
    "request_hash" VARCHAR(64) NOT NULL,
    "workflow_id" VARCHAR(255) NOT NULL,
    "workflow_run_id" VARCHAR(255) NOT NULL,
    "activity_id" VARCHAR(255) NOT NULL,
    "task_token" BYTEA NOT NULL,
    "response_id" VARCHAR(255),
    "model" VARCHAR(100),
    "status" VARCHAR(32) NOT NULL DEFAULT 'Pending',
    "failure_code" VARCHAR(100),
    "failure_message" TEXT,
    "submitted_at" BIGINT,
    "last_polled_at" BIGINT,
    "completed_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_ai_extractions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_ai_extractions_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id")
        REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_ai_extractions_document_extracted_at"
    ON "document_ai_extractions"("document_id", "organization_id", "business_unit_id", "extracted_at");

CREATE INDEX IF NOT EXISTS "idx_document_ai_extractions_status_poll"
    ON "document_ai_extractions"("status", "last_polled_at", "submitted_at");

CREATE INDEX IF NOT EXISTS "idx_document_ai_extractions_response_id"
    ON "document_ai_extractions"("response_id");
