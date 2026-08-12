-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250925214444_add_variables.tx.up.sql

CREATE TABLE IF NOT EXISTS "variable_formats"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "value_type" TEXT NOT NULL,
    "format_sql" TEXT NOT NULL,
    "is_active" INTEGER,
    "is_system" INTEGER,
    "version" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_variable_formats" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uk_format_name" UNIQUE ("organization_id", "name"),
    CONSTRAINT "fk_variable_formats_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_variable_formats_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_variable_formats_org" ON "variable_formats" ("organization_id", "value_type", "is_active");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_variable_formats_name" ON "variable_formats" ("name")WHERE "is_active" = true;

--bun:split

CREATE TABLE IF NOT EXISTS "variables"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "key" TEXT NOT NULL,
    "display_name" TEXT NOT NULL,
    "description" TEXT,
    "category" TEXT,
    "query" TEXT NOT NULL,
    "applies_to" TEXT NOT NULL,
    "required_params" TEXT,
    "default_value" TEXT,
    "format_id" TEXT,
    "value_type" TEXT,
    "is_active" INTEGER,
    "is_system" INTEGER,
    "is_validated" INTEGER,
    "tags" TEXT,
    "version" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_variables" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uk_variable_key" UNIQUE ("organization_id", "key"),
    CONSTRAINT "fk_variables_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_variables_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_variables_format" FOREIGN KEY ("format_id", "business_unit_id", "organization_id")
        REFERENCES "variable_formats"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_variables_query_length" CHECK (length("query") > 0),
    CONSTRAINT "chk_variables_key_length" CHECK (length("key") >= 2 AND length("key") <= 100)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_variables_context" ON "variables" ("organization_id", "applies_to", "is_active");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_variables_key" ON "variables" ("organization_id", "key")WHERE "is_active" = true;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_variables_business_unit" ON "variables" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_variables_created_updated" ON "variables" ("created_at", "updated_at");
