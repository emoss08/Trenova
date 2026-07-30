-- Organization-authored document and message templates.
--
-- This supersedes 20251125000001_add_document_templates, which created
-- document_templates and generated_documents but was never referenced by any Go,
-- TypeScript, or SQL in the repository. Those tables are therefore provably empty
-- in every environment, so dropping them is safe. A migration is immutable once
-- shipped, which is why this is a new forward migration rather than an edit.
--
-- The recreate also namespaces five enums the earlier migration claimed under
-- generic names (template_status_enum, page_size_enum, orientation_enum,
-- generation_status_enum, delivery_method_enum), any of which a future domain would
-- collide with.
DROP TABLE IF EXISTS "generated_documents";

--bun:split
DROP TABLE IF EXISTS "document_templates";

--bun:split
DROP TYPE IF EXISTS "delivery_method_enum";

--bun:split
DROP TYPE IF EXISTS "generation_status_enum";

--bun:split
DROP TYPE IF EXISTS "orientation_enum";

--bun:split
DROP TYPE IF EXISTS "page_size_enum";

--bun:split
DROP TYPE IF EXISTS "template_status_enum";

--bun:split
CREATE TYPE "document_template_version_status_enum" AS ENUM(
    'Draft',
    'Active',
    'Archived'
);

--bun:split
CREATE TYPE "document_template_page_size_enum" AS ENUM(
    'Letter',
    'A4',
    'Legal'
);

--bun:split
CREATE TYPE "document_template_orientation_enum" AS ENUM(
    'Portrait',
    'Landscape'
);

--bun:split
CREATE TYPE "generated_document_status_enum" AS ENUM(
    'Pending',
    'Processing',
    'Completed',
    'Failed'
);

--bun:split
CREATE TYPE "generated_document_delivery_enum" AS ENUM(
    'None',
    'Email',
    'Download',
    'Portal'
);

--bun:split
-- kind is varchar rather than a Postgres enum, deliberately: the Go registry in
-- internal/core/domain/documenttemplate is the single source of truth for which
-- kinds exist, and adding one must not require a migration. This diverges from the
-- repository's usual habit of modelling enums in the database.
CREATE TABLE IF NOT EXISTS "document_templates"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "kind" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(200) NOT NULL,
    "description" text,
    "is_org_default" boolean NOT NULL DEFAULT FALSE,
    "active_version_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "created_by_id" varchar(100),
    "updated_by_id" varchar(100),
    CONSTRAINT "pk_document_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_templates_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_document_templates_updated_by" FOREIGN KEY ("updated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_document_templates_kind" CHECK (length(TRIM(BOTH FROM "kind")) > 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_templates_code" ON "document_templates"("organization_id", "business_unit_id", lower("code"));

--bun:split
-- One organization default per kind, enforced here rather than only in the service:
-- two defaults would make which template a customer receives depend on row order.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_templates_org_default" ON "document_templates"("organization_id", "business_unit_id", "kind")
WHERE
    "is_org_default";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_document_templates_tenant" ON "document_templates"("organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_document_templates_kind" ON "document_templates"("organization_id", "business_unit_id", "kind");

--bun:split
ALTER TABLE "document_templates"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('english', COALESCE("description", '')), 'B')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_document_templates_search" ON "document_templates" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE document_templates IS 'Organization-authored templates for customer documents and outbound messages; kind references the Go registry, not a database enum';

--bun:split
-- Content lives on the version, not the template, so editing never touches what is
-- live. source_version_id records what a draft was copied from, which is what makes
-- a rollback a new version rather than a mutation of history.
CREATE TABLE IF NOT EXISTS "document_template_versions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "source_version_id" varchar(100),
    "version_number" bigint NOT NULL,
    "status" document_template_version_status_enum NOT NULL DEFAULT 'Draft',
    "subject" text,
    "body_html" text,
    "body_text" text,
    "css_content" text,
    "header_html" text,
    "footer_html" text,
    "page_size" document_template_page_size_enum,
    "orientation" document_template_orientation_enum,
    "margin_top" numeric(6, 2),
    "margin_bottom" numeric(6, 2),
    "margin_left" numeric(6, 2),
    "margin_right" numeric(6, 2),
    "content_hash" varchar(64) NOT NULL,
    "starter_hash" varchar(64),
    "publish_notes" text,
    "published_by_id" varchar(100),
    "published_at" bigint,
    "archived_by_id" varchar(100),
    "archived_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "created_by_id" varchar(100),
    "updated_by_id" varchar(100),
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
    -- A version must carry something renderable. An empty version that reached
    -- Active would send a blank page to a customer.
    CONSTRAINT "chk_document_template_versions_has_content" CHECK (COALESCE(NULLIF(TRIM(BOTH FROM "body_html"), ''), NULLIF(TRIM(BOTH FROM "body_text"), ''), NULLIF(TRIM(BOTH FROM "subject"), '')) IS NOT NULL),
    -- Publishing and archiving must record who and when, together.
    CONSTRAINT "chk_document_template_versions_published_pair" CHECK (("published_at" IS NULL) = ("published_by_id" IS NULL)),
    CONSTRAINT "chk_document_template_versions_active_published" CHECK ("status" <> 'Active' OR "published_at" IS NOT NULL),
    CONSTRAINT "chk_document_template_versions_archived_pair" CHECK ("status" = 'Archived' OR "archived_at" IS NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_versions_number" ON "document_template_versions"("organization_id", "business_unit_id", "template_id", "version_number");

--bun:split
-- One draft and one active version per template. Two drafts would make "the draft"
-- ambiguous in the editor; two actives would make rendering order-dependent.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_versions_one_draft" ON "document_template_versions"("organization_id", "business_unit_id", "template_id")
WHERE
    "status" = 'Draft';

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_versions_one_active" ON "document_template_versions"("organization_id", "business_unit_id", "template_id")
WHERE
    "status" = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_document_template_versions_tenant" ON "document_template_versions"("organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_document_template_versions_history" ON "document_template_versions"("template_id", "version_number" DESC);

--bun:split
COMMENT ON TABLE document_template_versions IS 'Immutable-once-published template content; editing a draft never changes what is live';

--bun:split
-- Added after both tables exist because the reference is circular: a template names
-- its live version and a version names its template.
ALTER TABLE "document_templates"
    ADD CONSTRAINT "fk_document_templates_active_version" FOREIGN KEY ("active_version_id", "organization_id", "business_unit_id") REFERENCES "document_template_versions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
-- A separate table rather than columns on customers: kinds grow without bound, so a
-- column per kind would be permanent schema churn, and an assignment needs its own
-- audit trail. Leaves room for location or shipment-type scoping later, which is the
-- shape detention_policies already uses.
CREATE TABLE IF NOT EXISTS "document_template_assignments"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "kind" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "customer_id" varchar(100) NOT NULL,
    "assigned_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_document_template_assignments" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_template_assignments_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "document_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_template_assignments_assigned_by" FOREIGN KEY ("assigned_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split
-- One assignment per customer per kind: two would make the resolved template
-- depend on row order.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_template_assignments_scope" ON "document_template_assignments"("organization_id", "business_unit_id", "kind", "customer_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_document_template_assignments_resolution" ON "document_template_assignments"("organization_id", "business_unit_id", "customer_id", "kind");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_document_template_assignments_template" ON "document_template_assignments"("template_id");

--bun:split
COMMENT ON TABLE document_template_assignments IS 'Per-customer whole-template overrides; resolution is assignment, then organization default, then the built-in';

--bun:split
-- template_version_id is what makes an output reproducible: the exact content that
-- produced these bytes can always be read back.
CREATE TABLE IF NOT EXISTS "generated_documents"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "kind" varchar(100) NOT NULL,
    "template_id" varchar(100),
    "template_version_id" varchar(100),
    "document_id" varchar(100),
    "reference_type" varchar(50) NOT NULL,
    "reference_id" varchar(100) NOT NULL,
    "file_name" varchar(255) NOT NULL,
    "file_path" varchar(500),
    "file_size" bigint NOT NULL DEFAULT 0,
    "mime_type" varchar(100) NOT NULL DEFAULT 'application/pdf',
    "checksum" varchar(64),
    "content_hash" varchar(64),
    "warnings" jsonb,
    "status" generated_document_status_enum NOT NULL DEFAULT 'Pending',
    "error_message" text,
    "generated_at" bigint,
    "generated_by_id" varchar(100),
    "delivery_method" generated_document_delivery_enum NOT NULL DEFAULT 'None',
    "delivered_at" bigint,
    "delivered_to" varchar(255),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_generated_documents" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_generated_documents_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_generated_documents_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- RESTRICT, not CASCADE: a version that produced a document a customer has
    -- received must not be deletable, or the output stops being reproducible.
    CONSTRAINT "fk_generated_documents_template_version" FOREIGN KEY ("template_version_id", "organization_id", "business_unit_id") REFERENCES "document_template_versions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_generated_documents_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id") REFERENCES "document_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_generated_documents_document" FOREIGN KEY ("document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_generated_documents_generated_by" FOREIGN KEY ("generated_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_generated_documents_completed" CHECK ("status" <> 'Completed' OR ("generated_at" IS NOT NULL AND "file_path" IS NOT NULL)),
    CONSTRAINT "chk_generated_documents_failed" CHECK ("status" <> 'Failed' OR "error_message" IS NOT NULL),
    CONSTRAINT "chk_generated_documents_delivered_pair" CHECK (("delivered_at" IS NULL) = ("delivered_to" IS NULL))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_generated_documents_tenant" ON "generated_documents"("organization_id", "business_unit_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_generated_documents_reference" ON "generated_documents"("reference_type", "reference_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_generated_documents_version" ON "generated_documents"("template_version_id")
WHERE
    "template_version_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_generated_documents_created" ON "generated_documents"("created_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_generated_documents_status" ON "generated_documents"("status");

--bun:split
ALTER TABLE "generated_documents"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("file_name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("reference_type", '')), 'B')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_generated_documents_search" ON "generated_documents" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE generated_documents IS 'One row per rendered document, naming the exact template version that produced it';

--bun:split
-- Detention notices snapshot their rendered content so the UI can show what was
-- sent. They previously stored only plain text and rebuilt HTML at send time; both
-- forms and the version responsible are now recorded.
ALTER TABLE "detention_notices"
    ADD COLUMN IF NOT EXISTS "body_html" text;

--bun:split
ALTER TABLE "detention_notices"
    ADD COLUMN IF NOT EXISTS "template_version_id" varchar(100);

--bun:split
ALTER TABLE "detention_notices"
    ADD COLUMN IF NOT EXISTS "pdf_document_id" varchar(100);

--bun:split
ALTER TABLE "detention_notices"
    ADD CONSTRAINT "fk_detention_notices_template_version" FOREIGN KEY ("template_version_id", "organization_id", "business_unit_id") REFERENCES "document_template_versions"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "detention_notices"
    ADD CONSTRAINT "fk_detention_notices_pdf_document" FOREIGN KEY ("pdf_document_id", "organization_id", "business_unit_id") REFERENCES "documents"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;
