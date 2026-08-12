-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260116035631_data_standardization_control.tx.up.sql

CREATE TABLE IF NOT EXISTS "data_entry_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code_case" TEXT NOT NULL DEFAULT 'Upper',
    "name_case" TEXT NOT NULL DEFAULT 'TitleCase',
    "email_case" TEXT NOT NULL DEFAULT 'Lower',
    "city_case" TEXT NOT NULL DEFAULT 'TitleCase',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_data_entry_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_data_entry_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_data_entry_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_data_entry_controls_organization" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_data_entry_controls_business_unit" ON "data_entry_controls" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_data_entry_controls_created_at" ON "data_entry_controls" ("created_at", "updated_at");
