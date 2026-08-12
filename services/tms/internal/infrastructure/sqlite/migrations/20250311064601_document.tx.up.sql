-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250311064601_document.tx.up.sql

CREATE TABLE IF NOT EXISTS "documents"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "original_name" TEXT NOT NULL,
    "file_size" INTEGER NOT NULL CHECK ("file_size" > 0),
    "file_type" TEXT NOT NULL,
    "storage_path" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "description" TEXT,
    "resource_id" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL,
    "expiration_date" INTEGER,
    "tags" TEXT,
    "is_public" INTEGER NOT NULL DEFAULT 0,
    "uploaded_by_id" TEXT NOT NULL,
    "approved_by_id" TEXT,
    "approved_at" INTEGER,
    "preview_storage_path" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_documents_uploaded_by" FOREIGN KEY ("uploaded_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_documents_approved_by" FOREIGN KEY ("approved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_resource" ON "documents" ("resource_type", "resource_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_expiration" ON "documents" ("expiration_date")WHERE
    "expiration_date" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_tenant" ON "documents" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_documents_uploaded_by" ON "documents" ("uploaded_by_id", "created_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_documents_active ON documents (created_at DESC)WHERE
    status NOT IN ('Archived', 'Expired', 'Rejected');
