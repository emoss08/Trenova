-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260326110000_add_document_upload_sessions.tx.up.sql

CREATE TABLE IF NOT EXISTS "document_upload_sessions"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "document_id" TEXT,
    "resource_id" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL,
    "document_type_id" TEXT,
    "original_name" TEXT NOT NULL,
    "content_type" TEXT NOT NULL,
    "file_size" INTEGER NOT NULL,
    "storage_path" TEXT NOT NULL,
    "storage_provider_upload_id" TEXT,
    "strategy" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Initiated',
    "description" TEXT,
    "tags" TEXT NOT NULL DEFAULT '{}',
    "uploaded_parts" TEXT NOT NULL DEFAULT '[]',
    "part_size" INTEGER NOT NULL DEFAULT 0,
    "failure_code" TEXT,
    "failure_message" TEXT,
    "expires_at" INTEGER NOT NULL,
    "last_activity_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_upload_sessions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_upload_sessions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_upload_sessions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_upload_sessions_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_tenant_status" ON "document_upload_sessions" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_resource" ON "document_upload_sessions" ("resource_type", "resource_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_upload_sessions_expires_at" ON "document_upload_sessions" ("expires_at");
