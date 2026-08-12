-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251125000001_add_document_templates.tx.up.sql

CREATE TABLE IF NOT EXISTS "document_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "document_type_id" TEXT NOT NULL,
    "html_content" TEXT NOT NULL,
    "css_content" TEXT,
    "header_html" TEXT,
    "footer_html" TEXT,
    "page_size" TEXT NOT NULL DEFAULT 'Letter',
    "orientation" TEXT NOT NULL DEFAULT 'Portrait',
    "margin_top" INTEGER NOT NULL DEFAULT 20,
    "margin_bottom" INTEGER NOT NULL DEFAULT 20,
    "margin_left" INTEGER NOT NULL DEFAULT 20,
    "margin_right" INTEGER NOT NULL DEFAULT 20,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "created_by_id" TEXT,
    "updated_by_id" TEXT,
    CONSTRAINT "pk_document_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_document_templates_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "fk_document_templates_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON DELETE SET NULL,
    CONSTRAINT "uq_document_templates_code" UNIQUE ("organization_id", "business_unit_id", "code")
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_templates_bu_org ON "document_templates" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_templates_created_updated ON "document_templates" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_templates_status ON "document_templates" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_templates_document_type ON "document_templates" ("document_type_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_document_templates_default ON "document_templates" ("is_default")WHERE
    "is_default" = TRUE;

--bun:split

CREATE TABLE IF NOT EXISTS "generated_documents"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "document_type_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "reference_type" TEXT NOT NULL,
    "reference_id" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "file_path" TEXT NOT NULL,
    "file_size" INTEGER NOT NULL,
    "mime_type" TEXT NOT NULL DEFAULT 'application/pdf',
    "checksum" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "error_message" TEXT,
    "generated_at" INTEGER,
    "generated_by_id" TEXT,
    "delivery_method" TEXT NOT NULL DEFAULT 'None',
    "delivered_at" INTEGER,
    "delivered_to" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_generated_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_generated_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_generated_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_generated_documents_document_type" FOREIGN KEY ("document_type_id", "organization_id", "business_unit_id") REFERENCES "document_types"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_generated_documents_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "document_templates"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_generated_documents_generated_by" FOREIGN KEY ("generated_by_id") REFERENCES "users"("id") ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_generated_documents_bu_org ON "generated_documents" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_generated_documents_created ON "generated_documents" ("created_at");

--bun:split

CREATE INDEX IF NOT EXISTS idx_generated_documents_status ON "generated_documents" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS idx_generated_documents_reference ON "generated_documents" ("reference_type", "reference_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_generated_documents_template ON "generated_documents" ("template_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_generated_documents_document_type ON "generated_documents" ("document_type_id");
