-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260518130000_edi_partner_setting_schemas.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_partner_setting_schemas"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT,
    "organization_id" TEXT,
    "document_type_id" TEXT,
    "standard" TEXT NOT NULL,
    "transaction_set" TEXT NOT NULL,
    "direction" TEXT NOT NULL,
    "x12_version" TEXT NOT NULL,
    "schema_version" INTEGER NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_partner_setting_schemas" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_partner_setting_schemas_document_type" FOREIGN KEY ("document_type_id") REFERENCES "edi_document_types"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partner_setting_schemas_scope"
    ON "edi_partner_setting_schemas" (
        COALESCE("organization_id", ''),
        COALESCE("business_unit_id", ''),
        COALESCE("document_type_id", ''),
        "standard",
        "transaction_set",
        "direction",
        "x12_version",
        "schema_version"
    );

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_schemas_lookup"
    ON "edi_partner_setting_schemas" ("standard", "transaction_set", "direction", "x12_version", "status", "schema_version" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_schemas_document_type"
    ON "edi_partner_setting_schemas" ("document_type_id", "standard", "transaction_set", "direction", "x12_version", "status", "schema_version" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "edi_partner_setting_fields"(
    "id" TEXT NOT NULL,
    "schema_id" TEXT NOT NULL,
    "path" TEXT NOT NULL,
    "label" TEXT NOT NULL,
    "description" TEXT,
    "data_type" TEXT NOT NULL,
    "required" INTEGER NOT NULL DEFAULT 0,
    "nullable" INTEGER NOT NULL DEFAULT 0,
    "default_value" TEXT,
    "allowed_values" TEXT NOT NULL DEFAULT '[]',
    "secret" INTEGER NOT NULL DEFAULT 0,
    "group_key" TEXT,
    "display_order" INTEGER NOT NULL DEFAULT 0,
    "validation_pattern" TEXT,
    "min_length" INTEGER NOT NULL DEFAULT 0,
    "max_length" INTEGER NOT NULL DEFAULT 0,
    "usage_notes" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_partner_setting_fields" PRIMARY KEY ("id"),
    CONSTRAINT "fk_edi_partner_setting_fields_schema" FOREIGN KEY ("schema_id") REFERENCES "edi_partner_setting_schemas"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_path"
    ON "edi_partner_setting_fields" ("schema_id", "path");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_group"
    ON "edi_partner_setting_fields" ("schema_id", "group_key", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_required"
    ON "edi_partner_setting_fields" ("schema_id", "required", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_partner_setting_fields_secret"
    ON "edi_partner_setting_fields" ("schema_id", "secret", "status");

--bun:split

ALTER TABLE "edi_partner_document_profiles" ADD COLUMN "partner_settings_schema_id" TEXT;

--bun:split

ALTER TABLE "edi_partner_document_profiles" ADD COLUMN "partner_settings_schema_version" INTEGER;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_partner_document_profiles_partner_settings_schema"
    ON "edi_partner_document_profiles" ("partner_settings_schema_id", "partner_settings_schema_version");
