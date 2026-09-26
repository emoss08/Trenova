-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006810_accounting_drift_findings.tx.up.sql

ALTER TABLE "accounting_connections" ADD COLUMN "drift_checked_at" INTEGER;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "drift_error_category" TEXT;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "drift_error_message" TEXT;

--bun:split

CREATE TABLE IF NOT EXISTS "accounting_drift_findings"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "connection_id" TEXT NOT NULL,
    "object_type" TEXT NOT NULL,
    "object_id" TEXT NOT NULL,
    "object_number" TEXT,
    "party_id" TEXT,
    "party_name" TEXT,
    "external_id" TEXT,
    "external_url" TEXT,
    "kind" TEXT NOT NULL,
    "currency_code" TEXT NOT NULL,
    "trenova_minor" INTEGER,
    "provider_minor" INTEGER,
    "difference_minor" INTEGER,
    "trenova_state" TEXT,
    "provider_state" TEXT,
    "detail" TEXT NOT NULL DEFAULT '[]',
    "provider_modified_at" INTEGER,
    "provider_modified_by" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "resolution" TEXT,
    "resolution_note" TEXT,
    "fix_object_type" TEXT,
    "fix_object_id" TEXT,
    "resolved_by_id" TEXT,
    "resolved_at" INTEGER,
    "detected_at" INTEGER NOT NULL,
    "last_seen_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_drift_findings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_drift_findings_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_drift_findings_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_drift_findings_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_drift_findings_resolved_by" FOREIGN KEY ("resolved_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_drift_findings_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication', 'CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment')),
    CONSTRAINT "ck_accounting_drift_findings_kind" CHECK ("kind" IN ('AmountMismatch', 'StatusMismatch', 'DeletedInProvider', 'VoidedInProvider', 'CustomerBalanceMismatch')),
    CONSTRAINT "ck_accounting_drift_findings_status" CHECK ("status" IN ('Open', 'Resolved', 'Dismissed')),
    CONSTRAINT "ck_accounting_drift_findings_resolution" CHECK ("resolution" IS NULL OR "resolution" IN ('PushedTrenovaValue', 'AdjustedTrenova', 'NoLongerDiffers', 'Dismissed')),
    CONSTRAINT "ck_accounting_drift_findings_closed" CHECK ("status" = 'Open' OR ("resolution" IS NOT NULL AND "resolved_at" IS NOT NULL)),
    CONSTRAINT "ck_accounting_drift_findings_dismissed" CHECK ("status" <> 'Dismissed' OR "resolution_note" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_drift_findings_open" ON "accounting_drift_findings" ("organization_id", "business_unit_id", "connection_id", "object_type", "object_id", "kind")WHERE "status" = 'Open';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_drift_findings_status" ON "accounting_drift_findings" ("organization_id", "business_unit_id", "connection_id", "status", "detected_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_drift_findings_object" ON "accounting_drift_findings" ("organization_id", "business_unit_id", "object_id");
