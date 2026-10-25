CREATE TABLE IF NOT EXISTS "document_parsing_rule_sets" (
    "id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "description" TEXT,
    "document_kind" VARCHAR(100) NOT NULL,
    "priority" INTEGER NOT NULL DEFAULT 100,
    "published_version_id" VARCHAR(100),
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_parsing_rule_sets" PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_parsing_rule_sets_name_kind_scope"
    ON "document_parsing_rule_sets"("organization_id", "business_unit_id", "document_kind", lower("name"));
CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_sets_kind"
    ON "document_parsing_rule_sets"("organization_id", "business_unit_id", "document_kind");

CREATE TABLE IF NOT EXISTS "document_parsing_rule_versions" (
    "id" VARCHAR(100) NOT NULL,
    "rule_set_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "version_number" INTEGER NOT NULL,
    "status" VARCHAR(20) NOT NULL DEFAULT 'Draft',
    "label" VARCHAR(255),
    "parser_mode" VARCHAR(50) NOT NULL DEFAULT 'merge_with_base',
    "match_config" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "rule_document" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "validation_summary" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "published_at" BIGINT,
    "published_by_id" VARCHAR(100),
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_parsing_rule_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_parsing_rule_versions_rule_set" FOREIGN KEY ("rule_set_id", "organization_id", "business_unit_id")
        REFERENCES "document_parsing_rule_sets"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_parsing_rule_versions_rule_set_version"
    ON "document_parsing_rule_versions"("rule_set_id", "organization_id", "business_unit_id", "version_number");
CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_versions_status"
    ON "document_parsing_rule_versions"("organization_id", "business_unit_id", "status");
CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_versions_rule_set"
    ON "document_parsing_rule_versions"("rule_set_id", "organization_id", "business_unit_id");

ALTER TABLE "document_parsing_rule_sets"
    ADD CONSTRAINT "fk_document_parsing_rule_sets_published_version" FOREIGN KEY ("published_version_id", "organization_id", "business_unit_id")
        REFERENCES "document_parsing_rule_versions"("id", "organization_id", "business_unit_id") ON DELETE SET NULL;

CREATE TABLE IF NOT EXISTS "document_parsing_rule_fixtures" (
    "id" VARCHAR(100) NOT NULL,
    "rule_set_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "name" VARCHAR(255) NOT NULL,
    "description" TEXT,
    "file_name" VARCHAR(255),
    "provider_fingerprint" VARCHAR(100),
    "text_snapshot" TEXT NOT NULL,
    "page_snapshots" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "assertions" JSONB NOT NULL DEFAULT '{}'::jsonb,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_document_parsing_rule_fixtures" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_parsing_rule_fixtures_rule_set" FOREIGN KEY ("rule_set_id", "organization_id", "business_unit_id")
        REFERENCES "document_parsing_rule_sets"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_fixtures_rule_set"
    ON "document_parsing_rule_fixtures"("rule_set_id", "organization_id", "business_unit_id");
