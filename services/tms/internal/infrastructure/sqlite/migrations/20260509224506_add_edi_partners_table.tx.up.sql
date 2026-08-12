-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260509224506_add_edi_partners_table.tx.up.sql

CREATE TABLE IF NOT EXISTS "edi_partners"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL DEFAULT 'External',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "internal_organization_id" TEXT,
    "customer_id" TEXT,
    "default_transport_id" TEXT,
    "default_mapping_profile_id" TEXT,
    "default_validation_profile_id" TEXT,
    "timezone" TEXT,
    "country" TEXT NOT NULL DEFAULT 'US',
    "contact_name" TEXT,
    "contact_email" TEXT,
    "contact_phone" TEXT,
    "enabled_for_inbound" INTEGER NOT NULL DEFAULT 1,
    "enabled_for_outbound" INTEGER NOT NULL DEFAULT 1,
    "settings" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_edi_partners" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_edi_partners_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_partners_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_edi_partners_customer" FOREIGN KEY ("customer_id", "business_unit_id", "organization_id") REFERENCES "customers"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_edi_partners_internal_org" FOREIGN KEY ("internal_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partners_code_org" ON "edi_partners" (lower("code"), "organization_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partners_name_org" ON "edi_partners" (lower("name"), "organization_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_edi_partners_internal_relationship_org_bu" ON "edi_partners" ("organization_id", "business_unit_id", "internal_organization_id")WHERE "kind" = 'Internal' AND "internal_organization_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_edi_partners_created_updated" ON "edi_partners" ("created_at", "updated_at");
