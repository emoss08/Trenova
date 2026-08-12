-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260902000000_document_templates.tx.up.sql

DROP TABLE IF EXISTS "generated_documents";

--bun:split

DROP TABLE IF EXISTS "document_templates";

--bun:split

CREATE TABLE IF NOT EXISTS "document_templates"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "is_org_default" INTEGER NOT NULL DEFAULT 0,
    "active_version_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "created_by_id" TEXT,
    "updated_by_id" TEXT,
    CONSTRAINT "pk_document_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_document_templates_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_document_templates_kind" CHECK (length(TRIM("kind")) > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_templates_code" ON "document_templates" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_templates_org_default" ON "document_templates" ("organization_id", "business_unit_id", "kind")WHERE
    "is_org_default";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_templates_tenant" ON "document_templates" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_templates_kind" ON "document_templates" ("organization_id", "business_unit_id", "kind");

--bun:split

CREATE TABLE IF NOT EXISTS "document_template_versions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "source_version_id" TEXT,
    "version_number" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "subject" TEXT,
    "body_html" TEXT,
    "body_text" TEXT,
    "css_content" TEXT,
    "header_html" TEXT,
    "footer_html" TEXT,
    "page_size" TEXT,
    "orientation" TEXT,
    "margin_top" NUMERIC,
    "margin_bottom" NUMERIC,
    "margin_left" NUMERIC,
    "margin_right" NUMERIC,
    "content_hash" TEXT NOT NULL,
    "starter_hash" TEXT,
    "publish_notes" TEXT,
    "published_by_id" TEXT,
    "published_at" INTEGER,
    "archived_by_id" TEXT,
    "archived_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "created_by_id" TEXT,
    "updated_by_id" TEXT,
    CONSTRAINT "pk_document_template_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_template_versions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_versions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_versions_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "document_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_versions_source" FOREIGN KEY ("source_version_id", "organization_id", "business_unit_id") REFERENCES "document_template_versions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_document_template_versions_published_by" FOREIGN KEY ("published_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_document_template_versions_archived_by" FOREIGN KEY ("archived_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_document_template_versions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_document_template_versions_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_document_template_versions_number" CHECK ("version_number" > 0),
    CONSTRAINT "chk_document_template_versions_margins" CHECK (("margin_top" IS NULL OR "margin_top" BETWEEN 0 AND 100) AND ("margin_bottom" IS NULL OR "margin_bottom" BETWEEN 0 AND 100) AND ("margin_left" IS NULL OR "margin_left" BETWEEN 0 AND 100) AND ("margin_right" IS NULL OR "margin_right" BETWEEN 0 AND 100)),
    CONSTRAINT "chk_document_template_versions_has_content" CHECK (COALESCE(NULLIF(TRIM("body_html"), ''), NULLIF(TRIM("body_text"), ''), NULLIF(TRIM("subject"), '')) IS NOT NULL),
    CONSTRAINT "chk_document_template_versions_published_pair" CHECK (("published_at" IS NULL) = ("published_by_id" IS NULL)),
    CONSTRAINT "chk_document_template_versions_active_published" CHECK ("status" <> 'Active' OR "published_at" IS NOT NULL),
    CONSTRAINT "chk_document_template_versions_archived_pair" CHECK ("status" = 'Archived' OR "archived_at" IS NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_versions_number" ON "document_template_versions" ("organization_id", "business_unit_id", "template_id", "version_number");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_versions_one_draft" ON "document_template_versions" ("organization_id", "business_unit_id", "template_id")WHERE
    "status" = 'Draft';

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_versions_one_active" ON "document_template_versions" ("organization_id", "business_unit_id", "template_id")WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_template_versions_tenant" ON "document_template_versions" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_template_versions_history" ON "document_template_versions" ("template_id", "version_number" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "document_template_assignments"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "template_id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "assigned_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_template_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_template_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "document_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_assigned_by" FOREIGN KEY ("assigned_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_assignments_scope" ON "document_template_assignments" ("organization_id", "business_unit_id", "kind", "customer_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_template_assignments_resolution" ON "document_template_assignments" ("organization_id", "business_unit_id", "customer_id", "kind");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_template_assignments_template" ON "document_template_assignments" ("template_id");

--bun:split

CREATE TABLE IF NOT EXISTS "generated_documents"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "template_id" TEXT,
    "template_version_id" TEXT,
    "document_id" TEXT,
    "reference_type" TEXT NOT NULL,
    "reference_id" TEXT NOT NULL,
    "file_name" TEXT NOT NULL,
    "file_path" TEXT,
    "file_size" INTEGER NOT NULL DEFAULT 0,
    "mime_type" TEXT NOT NULL DEFAULT 'application/pdf',
    "checksum" TEXT,
    "content_hash" TEXT,
    "warnings" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "error_message" TEXT,
    "generated_at" INTEGER,
    "generated_by_id" TEXT,
    "delivery_method" TEXT NOT NULL DEFAULT 'None',
    "delivered_at" INTEGER,
    "delivered_to" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_generated_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_generated_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_generated_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_generated_documents_template_version" FOREIGN KEY ("template_version_id", "organization_id", "business_unit_id") REFERENCES "document_template_versions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_generated_documents_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "document_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_generated_documents_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_generated_documents_generated_by" FOREIGN KEY ("generated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_generated_documents_completed" CHECK ("status" <> 'Completed' OR ("generated_at" IS NOT NULL AND "file_path" IS NOT NULL)),
    CONSTRAINT "chk_generated_documents_failed" CHECK ("status" <> 'Failed' OR "error_message" IS NOT NULL),
    CONSTRAINT "chk_generated_documents_delivered_pair" CHECK (("delivered_at" IS NULL) = ("delivered_to" IS NULL))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_generated_documents_tenant" ON "generated_documents" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_generated_documents_reference" ON "generated_documents" ("reference_type", "reference_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_generated_documents_version" ON "generated_documents" ("template_version_id")WHERE
    "template_version_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_generated_documents_created" ON "generated_documents" ("created_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_generated_documents_status" ON "generated_documents" ("status");

--bun:split

ALTER TABLE "detention_notices" ADD COLUMN "body_html" TEXT;

--bun:split

ALTER TABLE "detention_notices" ADD COLUMN "template_version_id" TEXT;

--bun:split

ALTER TABLE "detention_notices" ADD COLUMN "pdf_document_id" TEXT;
