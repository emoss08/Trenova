-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260518120000_edi_source_context_schema.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_source_context_schemas"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT,
    "organization_id" TEXT,
    "standard" TEXT NOT NULL,
    "transaction_set" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "x12_version" TEXT NOT NULL,
    "context_key" TEXT NOT NULL,
    "schema_version" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_source_context_schemas" PRIMARY KEY ("id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_source_context_schemas_scope"
    ON "edi_source_context_schemas" (
        COALESCE("organization_id", ''),
        COALESCE("business_unit_id", ''),
        "standard",
        "transaction_set",
        "direction",
        "x12_version",
        "context_key",
        "schema_version"
    );

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_source_context_schemas_lookup"
    ON "edi_source_context_schemas" ("standard", "transaction_set", "direction", "x12_version", "context_key", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "edi_source_context_fields"(
    "id" TEXT NOT NULL,
    "schema_id" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "source_kind" TEXT NOT NULL,
    "data_type" TEXT NOT NULL,
    "repeated" INTEGER NOT NULL DEFAULT 0,
    "repeat_path" TEXT,
    "parent_path" TEXT,
    "display_name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_source_context_fields" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_source_context_fields_schema" FOREIGN KEY ("schema_id") REFERENCES "edi_source_context_schemas"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_source_context_fields_path"
    ON "edi_source_context_fields" ("schema_id", "path", COALESCE("repeat_path", ''));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_source_context_fields_search"
    ON "edi_source_context_fields" ("schema_id", "source_kind", "status", "repeated");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_source_context_fields_prefix"
    ON "edi_source_context_fields" ("schema_id", "path");
