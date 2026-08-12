-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260122000000_custom_field_values.tx.up.sql

CREATE TABLE IF NOT EXISTS "custom_field_values"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "definition_id" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL,
    "resource_id" TEXT NOT NULL,
    "value" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_custom_field_values" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_cfv_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cfv_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cfv_definition" FOREIGN KEY ("definition_id", "organization_id", "business_unit_id")
        REFERENCES "custom_field_definitions"("id", "organization_id", "business_unit_id")
        ON DELETE CASCADE,
    CONSTRAINT "uq_cfv_resource_definition" UNIQUE (
        "organization_id", "business_unit_id", "resource_type", "resource_id", "definition_id"
    ),
    CONSTRAINT "chk_cfv_value_not_null" CHECK (value IS NOT NULL)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfv_resource" ON "custom_field_values" ("resource_type", "resource_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfv_definition" ON "custom_field_values" ("definition_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfv_tenant" ON "custom_field_values" ("organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cfv_resource_tenant" ON "custom_field_values" (
    "organization_id", "business_unit_id", "resource_type", "resource_id"
);
