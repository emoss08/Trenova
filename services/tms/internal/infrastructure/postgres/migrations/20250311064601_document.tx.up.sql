--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "document_status_enum" AS ENUM(
    'Draft',
    'Active',
    'Archived',
    'Expired',
    'Pending',
    'Rejected',
    'PendingApproval'
);

--bun:split
CREATE TABLE IF NOT EXISTS "documents"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "file_name" varchar(255) NOT NULL,
    "original_name" varchar(255) NOT NULL,
    "file_size" bigint NOT NULL CHECK ("file_size" > 0),
    "file_type" varchar(100) NOT NULL,
    "storage_path" varchar(500) NOT NULL,
    "status" document_status_enum NOT NULL DEFAULT 'Active',
    "description" text,
    "resource_id" varchar(100) NOT NULL,
    "resource_type" varchar(100) NOT NULL,
    "expiration_date" bigint,
    "tags" varchar(100)[] DEFAULT '{}',
    "is_public" boolean NOT NULL DEFAULT FALSE,
    "uploaded_by_id" varchar(100) NOT NULL,
    "approved_by_id" varchar(100),
    "approved_at" bigint,
    "preview_storage_path" varchar(500),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_documents_uploaded_by" FOREIGN KEY ("uploaded_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_documents_approved_by" FOREIGN KEY ("approved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_documents_expiration_date" CHECK ("expiration_date" IS NULL OR "expiration_date" > EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint)
);

COMMENT ON TABLE documents IS 'Centralized document repository for all entity-related documents';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_documents_resource" ON "documents"("resource_type", "resource_id");

CREATE INDEX IF NOT EXISTS "idx_documents_expiration" ON "documents"("expiration_date")
WHERE
    "expiration_date" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_documents_file_names" ON "documents" USING gin((file_name || ' ' || original_name) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS "idx_documents_tags" ON "documents" USING GIN("tags");

CREATE INDEX IF NOT EXISTS "idx_documents_tenant" ON "documents"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_documents_dates_brin" ON "documents" USING BRIN(created_at, updated_at, expiration_date) WITH (pages_per_range = 128);

CREATE INDEX IF NOT EXISTS "idx_documents_uploaded_by" ON "documents"("uploaded_by_id", "created_at");

--bun:split
ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("file_name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("original_name", '')), 'A') ||
        setweight(immutable_to_tsvector('english', COALESCE("description", '')), 'B') ||
        setweight(immutable_to_tsvector('simple', COALESCE(immutable_array_to_string("tags", ' '), '')), 'C')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_documents_search ON documents USING GIN(search_vector);

--bun:split
CREATE INDEX IF NOT EXISTS idx_documents_active ON documents(created_at DESC)
WHERE
    status NOT IN ('Archived', 'Expired', 'Rejected');

--bun:split
CREATE OR REPLACE FUNCTION fn_expire_documents()
    RETURNS integer
    AS $$
DECLARE
    expired_count integer;
BEGIN
    UPDATE
        documents
    SET
        status = 'Expired',
        updated_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
    WHERE
        expiration_date IS NOT NULL
        AND expiration_date <= EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
        AND status != 'Expired';
    GET DIAGNOSTICS expired_count = ROW_COUNT;
    RETURN expired_count;
END;
$$
LANGUAGE plpgsql;
