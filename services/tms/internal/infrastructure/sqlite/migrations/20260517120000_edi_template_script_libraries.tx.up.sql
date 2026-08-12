-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260517120000_edi_template_script_libraries.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_template_script_libraries"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "template_version_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "language" TEXT NOT NULL DEFAULT 'Starlark',
    "script" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_template_script_libraries" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_template_script_libraries_version" FOREIGN KEY ("template_version_id", "business_unit_id", "organization_id") REFERENCES "edi_template_versions"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_template_script_libraries_name"
    ON "edi_template_script_libraries" ("template_version_id", "business_unit_id", "organization_id", lower("name"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_template_script_libraries_version"
    ON "edi_template_script_libraries" ("template_version_id", "business_unit_id", "organization_id", "status");
