--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "document_classification_enum" AS ENUM (
    'Public',
    'Private',
    'Sensitive',
    'Regulatory'
);

CREATE TYPE "document_category_enum" AS ENUM (
    'Shipment',
    'Worker',
    'Regulatory',
    'Profile',
    'Branding',
    'Invoice',
    'Contract',
    'Other'
);

CREATE TABLE IF NOT EXISTS "document_types" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "code" varchar(10) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "color" varchar(10),
    -- Bucket Metadata
    "document_classification" document_classification_enum NOT NULL DEFAULT 'Public',
    "document_category" document_category_enum NOT NULL DEFAULT 'Other',
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_document_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_document_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for document_types table
CREATE UNIQUE INDEX "idx_document_types_code" ON "document_types" (lower("code"), "organization_id");

CREATE INDEX "idx_document_types_name" ON "document_types" ("name");

CREATE INDEX "idx_document_types_business_unit_organization" ON "document_types" ("business_unit_id", "organization_id");

CREATE INDEX "idx_document_types_created_updated" ON "document_types" ("created_at", "updated_at");

COMMENT ON TABLE "document_types" IS 'Stores information about document types';

--bun:split
ALTER TABLE "document_types"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_types_search ON document_types USING GIN (search_vector);

--bun:split
CREATE OR REPLACE FUNCTION document_types_search_vector_update ()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.name, '')), 'B');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS document_types_search_vector_trigger ON document_types;

--bun:split
CREATE TRIGGER document_types_search_vector_trigger
    BEFORE INSERT OR UPDATE ON document_types
    FOR EACH ROW
    EXECUTE FUNCTION document_types_search_vector_update ();

--bun:split
ALTER TABLE document_types
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE document_types
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_types_trgm_code ON document_types USING gin (code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_types_trgm_name ON document_types USING gin (name gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_types_trgm_code_name ON document_types USING gin ((code || ' ' || name) gin_trgm_ops);

--bun:split
ALTER TABLE "documents"
    ADD COLUMN IF NOT EXISTS document_type_id varchar(100) NOT NULL;

--bun:split
ALTER TABLE "documents"
    ADD CONSTRAINT "fk_documents_document_type" FOREIGN KEY ("document_type_id", "business_unit_id", "organization_id") REFERENCES "document_types" ("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE;
