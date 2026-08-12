-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250405000244_customer_email_profile.tx.up.sql

CREATE TABLE IF NOT EXISTS "customer_email_profiles"(
    "id" TEXT NOT NULL,
    "customer_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "subject" TEXT,
    "comment" TEXT,
    "from_email" TEXT,
    "to_recipients" TEXT,
    "cc_recipients" TEXT,
    "bcc_recipients" TEXT,
    "read_receipt" INTEGER NOT NULL DEFAULT 0,
    "attachment_name" TEXT,
    "send_invoice_on_generation" INTEGER NOT NULL DEFAULT 1,
    "include_shipment_detail" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_customer_email_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id", "customer_id"),
    CONSTRAINT "fk_customer_email_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_email_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customer_email_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_customer_email_profiles_customer" UNIQUE ("customer_id", "organization_id", "business_unit_id"),
    CONSTRAINT "uq_customer_email_profiles_id_org_bu" UNIQUE ("id", "organization_id", "business_unit_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_customer_id" ON "customer_email_profiles" ("customer_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_bu_org" ON "customer_email_profiles" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_created_updated" ON "customer_email_profiles" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_customer_email_profiles_from_email" ON "customer_email_profiles" ("from_email", "organization_id", "business_unit_id")WHERE "from_email" IS NOT NULL;
