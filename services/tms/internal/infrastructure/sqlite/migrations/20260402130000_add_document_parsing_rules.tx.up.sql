-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260402130000_add_document_parsing_rules.tx.up.sql

CREATE TABLE IF NOT EXISTS "document_parsing_rule_sets" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "document_kind" TEXT NOT NULL,
    "priority" INTEGER NOT NULL DEFAULT 100,
    "published_version_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_parsing_rule_sets" PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_parsing_rule_sets_name_kind_scope"
    ON "document_parsing_rule_sets" ("organization_id", "business_unit_id", "document_kind", lower("name"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_sets_kind"
    ON "document_parsing_rule_sets" ("organization_id", "business_unit_id", "document_kind");

--bun:split

CREATE TABLE IF NOT EXISTS "document_parsing_rule_versions" (
    "id" TEXT NOT NULL,
    "rule_set_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "version_number" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "label" TEXT,
    "parser_mode" TEXT NOT NULL DEFAULT 'merge_with_base',
    "match_config" TEXT NOT NULL DEFAULT '{}',
    "rule_document" TEXT NOT NULL DEFAULT '{}',
    "validation_summary" TEXT NOT NULL DEFAULT '{}',
    "published_at" INTEGER,
    "published_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_parsing_rule_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_parsing_rule_versions_rule_set" FOREIGN KEY ("rule_set_id", "organization_id", "business_unit_id")
        REFERENCES "document_parsing_rule_sets"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_document_parsing_rule_versions_rule_set_version"
    ON "document_parsing_rule_versions" ("rule_set_id", "organization_id", "business_unit_id", "version_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_versions_status"
    ON "document_parsing_rule_versions" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_versions_rule_set"
    ON "document_parsing_rule_versions" ("rule_set_id", "organization_id", "business_unit_id");

--bun:split

CREATE TABLE IF NOT EXISTS "document_parsing_rule_fixtures" (
    "id" TEXT NOT NULL,
    "rule_set_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "file_name" TEXT,
    "provider_fingerprint" TEXT,
    "text_snapshot" TEXT NOT NULL,
    "page_snapshots" TEXT NOT NULL DEFAULT '[]',
    "assertions" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_parsing_rule_fixtures" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_document_parsing_rule_fixtures_rule_set" FOREIGN KEY ("rule_set_id", "organization_id", "business_unit_id")
        REFERENCES "document_parsing_rule_sets"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_parsing_rule_fixtures_rule_set"
    ON "document_parsing_rule_fixtures" ("rule_set_id", "organization_id", "business_unit_id");
