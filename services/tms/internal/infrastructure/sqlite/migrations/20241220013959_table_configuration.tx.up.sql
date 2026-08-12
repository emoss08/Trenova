-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241220013959_table_configuration.tx.up.sql

CREATE TABLE IF NOT EXISTS "table_configurations"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "resource" TEXT NOT NULL,
    "table_config" TEXT NOT NULL,
    "visibility" TEXT NOT NULL DEFAULT 'Private',
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_table_configurations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_table_configurations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_configurations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_configurations_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_table_configurations_default" ON "table_configurations" ("user_id", "resource", "is_default")WHERE
    "is_default" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_table_configurations_business_unit" ON "table_configurations" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_table_configurations_organization" ON "table_configurations" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_table_configurations_user_id" ON "table_configurations" ("user_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_table_configurations_resource" ON "table_configurations" ("resource");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_table_configurations_visibility" ON "table_configurations" ("visibility");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_table_configurations_created_updated" ON "table_configurations" ("created_at", "updated_at");

--bun:split

CREATE TABLE IF NOT EXISTS "table_configuration_shares"(
    "id" TEXT NOT NULL,
    "configuration_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shared_with_id" TEXT NOT NULL,
    "share_type" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_table_configuration_shares" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_configuration_shares_config" FOREIGN KEY ("configuration_id", "organization_id", "business_unit_id") REFERENCES "table_configurations"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_configuration_shares" UNIQUE ("configuration_id", "shared_with_id", "share_type")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_configuration_shares_config" ON "table_configuration_shares" ("configuration_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_configuration_shares_shared_with" ON "table_configuration_shares" ("shared_with_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_configuration_shares_type" ON "table_configuration_shares" ("share_type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_configuration_shares_created" ON "table_configuration_shares" ("created_at");
