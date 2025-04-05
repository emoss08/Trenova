CREATE TYPE "document_type_enum" AS ENUM(
    'License', -- Driver's license, business license, etc.
    'Registration', -- Vehicle registration
    'Insurance', -- Insurance documents
    'Invoice', -- Customer invoices
    'ProofOfDelivery', -- POD documents
    'BillOfLading', -- BOL documents
    'DriverLog', -- Driver HOS logs
    'MedicalCertificate', -- Driver medical certificates
    'Contract', -- Business contracts
    'Maintenance', -- Maintenance records
    'AccidentReport', -- Accident or incident reports
    'TrainingRecord', -- Driver or employee training documents
    'Other' -- Miscellaneous documents
);

--bun:split
CREATE TYPE "document_status_enum" AS ENUM(
    'Draft', -- Document in draft state
    'Active', -- Document is active and valid
    'Archived', -- Document has been archived
    'Expired', -- Document has expired
    'Pending', -- Document is pending
    'Rejected', -- Document was rejected during approval
    'PendingApproval' -- Document is awaiting approval
);

--bun:split
CREATE TABLE IF NOT EXISTS "documents"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- File metadata
    "file_name" varchar(255) NOT NULL,
    "original_name" varchar(255) NOT NULL,
    "file_size" bigint NOT NULL CHECK ("file_size" > 0),
    "file_type" varchar(100) NOT NULL,
    "storage_path" varchar(500) NOT NULL,
    -- Document classification
    "document_type" document_type_enum NOT NULL,
    "status" document_status_enum NOT NULL DEFAULT 'Active',
    "description" text,
    -- Entity association (polymorphic relationship)
    "resource_id" varchar(100) NOT NULL,
    "resource_type" varchar(100) NOT NULL,
    -- Additional metadata
    "expiration_date" bigint,
    "tags" varchar(100)[] DEFAULT '{}',
    "is_public" boolean NOT NULL DEFAULT FALSE,
    -- Audit fields
    "uploaded_by_id" varchar(100) NOT NULL,
    "approved_by_id" varchar(100),
    "approved_at" bigint,
    -- Standard metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_documents_uploaded_by" FOREIGN KEY ("uploaded_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_documents_approved_by" FOREIGN KEY ("approved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    -- Check expiration_date is in the future when inserted/updated
    CONSTRAINT "chk_documents_expiration_date" CHECK ("expiration_date" IS NULL OR "expiration_date" > EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint)
);

COMMENT ON TABLE documents IS 'Centralized document repository for all entity-related documents';

--bun:split
-- Create required indexes for performance
-- Index for polymorphic relationship queries (the most common query pattern)
CREATE INDEX IF NOT EXISTS "idx_documents_resource" ON "documents"("resource_type", "resource_id");

-- Index for document classification
CREATE INDEX IF NOT EXISTS "idx_documents_type_status" ON "documents"("document_type", "status");

-- Index for expiration tracking (compliance management)
CREATE INDEX IF NOT EXISTS "idx_documents_expiration" ON "documents"("expiration_date")
WHERE
    "expiration_date" IS NOT NULL;

-- Index for text search on file names
CREATE INDEX IF NOT EXISTS "idx_documents_file_names" ON "documents" USING gin((file_name || ' ' || original_name) gin_trgm_ops);

-- Index for tags (using GIN for array operations)
CREATE INDEX IF NOT EXISTS "idx_documents_tags" ON "documents" USING GIN("tags");

-- Index for multi-tenant queries
CREATE INDEX IF NOT EXISTS "idx_documents_tenant" ON "documents"("business_unit_id", "organization_id");

-- BRIN index for date-based operations (efficient for time-series data)
CREATE INDEX IF NOT EXISTS "idx_documents_dates_brin" ON "documents" USING BRIN(created_at, updated_at, expiration_date) WITH (pages_per_range = 128);

-- Index for upload auditing
CREATE INDEX IF NOT EXISTS "idx_documents_uploaded_by" ON "documents"("uploaded_by_id", "created_at");

--bun:split
-- Add search vector for full-text search
ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
-- Create GIN index on search vector
CREATE INDEX IF NOT EXISTS idx_documents_search ON documents USING GIN(search_vector);

--bun:split
-- Setup trigger for automatic search vector updates
CREATE OR REPLACE FUNCTION documents_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.file_name, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.original_name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B') || setweight(to_tsvector('english', COALESCE(CAST(NEW.document_type AS text), '')), 'B') || setweight(to_tsvector('english', COALESCE(array_to_string(NEW.tags, ' '), '')), 'C');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    -- Auto-expire documents
    IF NEW.expiration_date IS NOT NULL AND NEW.expiration_date <= EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint AND NEW.status != 'Expired' THEN
        NEW.status := 'Expired';
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
-- Create trigger for search vector updates
DROP TRIGGER IF EXISTS documents_search_vector_trigger ON documents;

--bun:split
CREATE TRIGGER documents_search_vector_trigger
    BEFORE INSERT OR UPDATE ON documents
    FOR EACH ROW
    EXECUTE FUNCTION documents_search_vector_update();

--bun:split
-- Create special index for active documents
CREATE INDEX IF NOT EXISTS idx_documents_active ON documents(created_at DESC)
WHERE
    status NOT IN ('Archived', 'Expired', 'Rejected');

--bun:split
-- Optimize statistics collection for commonly filtered columns
ALTER TABLE documents
    ALTER COLUMN status SET STATISTICS 1000;

ALTER TABLE documents
    ALTER COLUMN document_type SET STATISTICS 1000;

ALTER TABLE documents
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE documents
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE documents
    ALTER COLUMN resource_type SET STATISTICS 1000;

-- Create view for expired documents (compliance reporting)
-- CREATE OR REPLACE VIEW vw_expired_documents AS
-- SELECT
--     d.*,
--     o.name AS organization_name,
--     bu.name AS business_unit_name,
--     u.name AS uploaded_by_name
-- FROM
--     documents d
--     JOIN organizations o ON d.organization_id = o.id
--     JOIN business_units bu ON d.business_unit_id = bu.id
--     JOIN users u ON d.uploaded_by_id = u.id
-- WHERE
--     d.expiration_date IS NOT NULL
--     AND d.expiration_date <= EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint
--     AND d.status != 'Expired';

-- Create view for documents needing approval
-- CREATE OR REPLACE VIEW vw_pending_approval_documents AS
-- SELECT
--     d.*,
--     o.name AS organization_name,
--     bu.name AS business_unit_name,
--     u.name AS uploaded_by_name
-- FROM
--     documents d
--     JOIN organizations o ON d.organization_id = o.id
--     JOIN business_units bu ON d.business_unit_id = bu.id
--     JOIN users u ON d.uploaded_by_id = u.id
-- WHERE
--     d.status = 'PendingApproval';

--bun:split
-- Create function to set document status to expired automatically
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

