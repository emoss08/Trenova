-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006620_accounting_mappings.tx.up.sql

ALTER TABLE "accounting_connections" ADD COLUMN "setup_step" TEXT NOT NULL DEFAULT 'Mappings';

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "reference_refresh_started_at" INTEGER;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "reference_refreshed_at" INTEGER;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "reference_refresh_error" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS "accounting_reference_objects"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "connection_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "external_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "search_name" TEXT NOT NULL,
    "fully_qualified_name" TEXT,
    "number" TEXT,
    "description" TEXT,
    "classification" TEXT,
    "account_type" TEXT,
    "account_sub_type" TEXT,
    "item_type" TEXT,
    "sub_type" TEXT,
    "parent_external_id" TEXT,
    "active" INTEGER NOT NULL DEFAULT 1,
    "currency_code" TEXT,
    "sync_token" TEXT,
    "company_name" TEXT,
    "email" TEXT,
    "address_line1" TEXT,
    "city" TEXT,
    "state" TEXT,
    "postal_code" TEXT,
    "income_account_external_id" TEXT,
    "due_days" INTEGER,
    "is_1099" INTEGER NOT NULL DEFAULT 0,
    "provider_updated_at" INTEGER,
    "last_seen_at" INTEGER NOT NULL,
    "removed_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_reference_objects" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_reference_objects_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_reference_objects_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_reference_objects_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_accounting_reference_objects_kind" CHECK ("kind" IN ('Account', 'Item', 'Customer', 'Vendor', 'Term', 'PaymentMethod'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_reference_objects_external" ON "accounting_reference_objects" ("organization_id", "business_unit_id", "connection_id", "kind", "external_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_reference_objects_search" ON "accounting_reference_objects" ("organization_id", "business_unit_id", "connection_id", "kind", "search_name");

--bun:split

CREATE TABLE IF NOT EXISTS "accounting_mappings"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "connection_id" TEXT NOT NULL,
    "target_type" TEXT NOT NULL,
    "trenova_object_id" TEXT,
    "trenova_key" TEXT,
    "target_label" TEXT NOT NULL,
    "search_label" TEXT NOT NULL,
    "provider_kind" TEXT NOT NULL,
    "external_id" TEXT,
    "external_name" TEXT,
    "state" TEXT NOT NULL DEFAULT 'Unmatched',
    "source" TEXT,
    "confidence" REAL,
    "reason" TEXT,
    "signals" TEXT NOT NULL DEFAULT '{}',
    "confirmed_by_id" TEXT,
    "confirmed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_mappings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_mappings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_mappings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_mappings_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_mappings_confirmed_by" FOREIGN KEY ("confirmed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_mappings_target_type" CHECK ("target_type" IN ('AccountRole', 'LineType', 'AccessorialCharge', 'ItemRole', 'Customer', 'Carrier', 'PaymentTerm', 'PaymentMethod')),
    CONSTRAINT "ck_accounting_mappings_provider_kind" CHECK ("provider_kind" IN ('Account', 'Item', 'Customer', 'Vendor', 'Term', 'PaymentMethod')),
    CONSTRAINT "ck_accounting_mappings_state" CHECK ("state" IN ('Unmatched', 'Proposed', 'Confirmed')),
    CONSTRAINT "ck_accounting_mappings_source" CHECK ("source" IS NULL OR "source" IN ('Suggested', 'Model', 'Manual', 'CreatedInProvider', 'Agent')),
    CONSTRAINT "ck_accounting_mappings_target_identity" CHECK (("trenova_object_id" IS NULL) <> ("trenova_key" IS NULL)),
    CONSTRAINT "ck_accounting_mappings_external_when_matched" CHECK (("state" = 'Unmatched') = ("external_id" IS NULL)),
    CONSTRAINT "ck_accounting_mappings_confidence" CHECK ("confidence" IS NULL OR ("confidence" >= 0 AND "confidence" <= 1))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_mappings_target" ON "accounting_mappings" ("organization_id", "business_unit_id", "connection_id", "target_type", COALESCE("trenova_object_id", ''), COALESCE("trenova_key", ''));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_mappings_review" ON "accounting_mappings" ("organization_id", "business_unit_id", "connection_id", "target_type", "state");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_mappings_external" ON "accounting_mappings" ("organization_id", "business_unit_id", "connection_id", "provider_kind", "external_id")WHERE "external_id" IS NOT NULL;
