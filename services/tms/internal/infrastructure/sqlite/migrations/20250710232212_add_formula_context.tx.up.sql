-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250710232212_add_formula_context.tx.up.sql

CREATE TABLE IF NOT EXISTS "formula_contexts" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "context_type" TEXT NOT NULL DEFAULT 'CUSTOM',
    "data_source" TEXT,
    "value_type" TEXT NOT NULL,
    "validation_rules" TEXT,
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_formula_contexts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_formula_contexts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_formula_contexts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_formula_contexts_custom_data_source" CHECK (("context_type" = 'CUSTOM' AND "data_source" IS NOT NULL) OR
        ("context_type" = 'BUILT_IN')),
    CONSTRAINT "chk_formula_contexts_name_length" CHECK (length("name") > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_formula_contexts_name_unique"
ON "formula_contexts" (lower("name"), "organization_id")WHERE "is_system" = FALSE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_contexts_organization"
ON "formula_contexts" ("organization_id", "context_type", "name");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_contexts_business_unit"
ON "formula_contexts" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_contexts_system"
ON "formula_contexts" ("is_system", "context_type")WHERE "is_system" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_contexts_created_updated"
ON "formula_contexts" ("created_at", "updated_at");

--bun:split

CREATE TABLE IF NOT EXISTS "formula_schemas" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "schema_uri" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT,
    "type" TEXT NOT NULL DEFAULT 'object',
    "version" TEXT NOT NULL,
    "properties" TEXT NOT NULL,
    "required" TEXT,
    "data_source" TEXT,
    "formula_context" TEXT,
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_formula_schemas" PRIMARY KEY ("id", "organization_id"),
    CONSTRAINT "fk_formula_schemas_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_formula_schemas_uri" UNIQUE ("schema_uri", "organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_schemas_organization"
ON "formula_schemas" ("organization_id", "is_active");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_schemas_system"
ON "formula_schemas" ("is_system")WHERE "is_system" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_formula_schemas_created_updated"
ON "formula_schemas" ("created_at", "updated_at");
