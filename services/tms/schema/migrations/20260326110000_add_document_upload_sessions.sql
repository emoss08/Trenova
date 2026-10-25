CREATE TYPE "document_upload_session_status_enum" AS ENUM(
    'Initiated',
    'Uploading',
    'Paused',
    'Completing',
    'Completed',
    'Failed',
    'Canceled',
    'Expired'
);

CREATE TABLE IF NOT EXISTS "document_upload_sessions"(
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "document_id" VARCHAR(100),
    "resource_id" VARCHAR(100) NOT NULL,
    "resource_type" VARCHAR(100) NOT NULL,
    "document_type_id" VARCHAR(100),
    "original_name" VARCHAR(255) NOT NULL,
    "content_type" VARCHAR(255) NOT NULL,
    "file_size" BIGINT NOT NULL,
    "storage_path" VARCHAR(500) NOT NULL,
    "storage_provider_upload_id" VARCHAR(255),
    "strategy" VARCHAR(20) NOT NULL,
    "status" document_upload_session_status_enum NOT NULL DEFAULT 'Initiated',
    "description" TEXT,
    "tags" VARCHAR(100)[] NOT NULL DEFAULT '{}',
    "uploaded_parts" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "part_size" BIGINT NOT NULL DEFAULT 0,
    "failure_code" VARCHAR(100),
    "failure_message" TEXT,
    "expires_at" BIGINT NOT NULL,
    "last_activity_at" BIGINT NOT NULL,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_upload_sessions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_upload_sessions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_upload_sessions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_upload_sessions_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_tenant_status" ON "document_upload_sessions"("organization_id", "business_unit_id", "status");
CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_resource" ON "document_upload_sessions"("resource_type", "resource_id");
CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_expires_at" ON "document_upload_sessions"("expires_at");
