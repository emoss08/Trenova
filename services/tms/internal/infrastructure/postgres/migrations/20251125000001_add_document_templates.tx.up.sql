CREATE TYPE template_status_enum AS ENUM(
    'Draft',
    'Active',
    'Archived'
);

--bun:split
CREATE TYPE page_size_enum AS ENUM(
    'Letter',
    'A4',
    'Legal'
);

--bun:split
CREATE TYPE orientation_enum AS ENUM(
    'Portrait',
    'Landscape'
);

--bun:split
CREATE TYPE generation_status_enum AS ENUM(
    'Pending',
    'Processing',
    'Completed',
    'Failed'
);

--bun:split
CREATE TYPE delivery_method_enum AS ENUM(
    'None',
    'Email',
    'Download',
    'Portal'
);

--bun:split
CREATE TABLE IF NOT EXISTS "document_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "document_type_id" varchar(100) NOT NULL,
    "html_content" text NOT NULL,
    "css_content" text,
    "header_html" text,
    "footer_html" text,
    "page_size" page_size_enum NOT NULL DEFAULT 'Letter',
    "orientation" orientation_enum NOT NULL DEFAULT 'Portrait',
    "margin_top" int NOT NULL DEFAULT 20,
    "margin_bottom" int NOT NULL DEFAULT 20,
    "margin_left" int NOT NULL DEFAULT 20,
    "margin_right" int NOT NULL DEFAULT 20,
    "status" template_status_enum NOT NULL DEFAULT 'Draft',
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "is_system" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    "created_by_id" varchar(100),
    "updated_by_id" varchar(100),
    CONSTRAINT "pk_document_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_document_templates_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_document_templates_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_document_templates_code" UNIQUE ("organization_id", "business_unit_id", "code")
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_templates_bu_org ON "document_templates"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_templates_created_updated ON "document_templates"("created_at", "updated_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_templates_status ON "document_templates"("status");

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_templates_document_type ON "document_templates"("document_type_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_templates_default ON "document_templates"("is_default")
WHERE
    "is_default" = TRUE;

--bun:split
ALTER TABLE "document_templates"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_document_templates_search_vector ON "document_templates" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION document_templates_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS document_templates_search_update ON "document_templates";

--bun:split
CREATE TRIGGER document_templates_search_update
    BEFORE INSERT OR UPDATE ON "document_templates"
    FOR EACH ROW
    EXECUTE FUNCTION document_templates_search_trigger();

--bun:split
CREATE TABLE IF NOT EXISTS "generated_documents"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "document_type_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "reference_type" varchar(50) NOT NULL,
    "reference_id" varchar(100) NOT NULL,
    "file_name" varchar(255) NOT NULL,
    "file_path" varchar(500) NOT NULL,
    "file_size" bigint NOT NULL,
    "mime_type" varchar(100) NOT NULL DEFAULT 'application/pdf',
    "checksum" varchar(64),
    "status" generation_status_enum NOT NULL DEFAULT 'Pending',
    "error_message" text,
    "generated_at" bigint,
    "generated_by_id" varchar(100),
    "delivery_method" delivery_method_enum NOT NULL DEFAULT 'None',
    "delivered_at" bigint,
    "delivered_to" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp)::bigint,
    CONSTRAINT "pk_generated_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_generated_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_generated_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_generated_documents_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_generated_documents_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "document_templates"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_generated_documents_generated_by" FOREIGN KEY ("generated_by_id") REFERENCES "users"("id") ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_generated_documents_bu_org ON "generated_documents"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_generated_documents_created ON "generated_documents"("created_at");

--bun:split
CREATE INDEX IF NOT EXISTS idx_generated_documents_status ON "generated_documents"("status");

--bun:split
CREATE INDEX IF NOT EXISTS idx_generated_documents_reference ON "generated_documents"("reference_type", "reference_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_generated_documents_template ON "generated_documents"("template_id");

--bun:split
CREATE INDEX IF NOT EXISTS idx_generated_documents_document_type ON "generated_documents"("document_type_id");

--bun:split
ALTER TABLE "generated_documents"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_generated_documents_search_vector ON "generated_documents" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION generated_documents_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.file_name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.reference_type, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS generated_documents_search_update ON "generated_documents";

--bun:split
CREATE TRIGGER generated_documents_search_update
    BEFORE INSERT OR UPDATE ON "generated_documents"
    FOR EACH ROW
    EXECUTE FUNCTION generated_documents_search_trigger();
