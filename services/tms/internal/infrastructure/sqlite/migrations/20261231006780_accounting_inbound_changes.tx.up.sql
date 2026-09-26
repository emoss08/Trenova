-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006780_accounting_inbound_changes.tx.up.sql

ALTER TABLE "accounting_connections" ADD COLUMN "change_cursor" TEXT;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "changes_read_at" INTEGER;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "changes_error_category" TEXT;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "changes_error_message" TEXT;

--bun:split

ALTER TABLE "accounting_connections" ADD COLUMN "inbound_payment_policy" TEXT NOT NULL DEFAULT 'Propose';

--bun:split

CREATE TABLE IF NOT EXISTS "accounting_inbound_changes"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "connection_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "external_id" TEXT NOT NULL,
    "external_number" TEXT,
    "external_url" TEXT,
    "provider_modified_at" INTEGER,
    "provider_modified_by" TEXT,
    "txn_date" INTEGER NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "currency_code" TEXT NOT NULL,
    "party_external_id" TEXT,
    "party_name" TEXT,
    "party_object_id" TEXT,
    "document" TEXT NOT NULL DEFAULT '{}',
    "status" TEXT NOT NULL DEFAULT 'Detected',
    "reason" TEXT,
    "resolution" TEXT,
    "applied_objects" TEXT NOT NULL DEFAULT '[]',
    "decided_by_id" TEXT,
    "decided_at" INTEGER,
    "note" TEXT,
    "detected_at" INTEGER NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_inbound_changes" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_inbound_changes_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_inbound_changes_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_inbound_changes_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_inbound_changes_decided_by" FOREIGN KEY ("decided_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_inbound_changes_kind" CHECK ("kind" IN ('CustomerPayment', 'BillPayment')),
    CONSTRAINT "ck_accounting_inbound_changes_status" CHECK ("status" IN ('Detected', 'Proposed', 'Applied', 'Ignored', 'Superseded')),
    CONSTRAINT "ck_accounting_inbound_changes_reason" CHECK ("reason" IS NULL OR "reason" IN ('PolicyPropose', 'PeriodNotOpen', 'UnknownDocument', 'PartyMismatch', 'Overpayment', 'PartialBillPayment', 'AlreadyPaid', 'CurrencyMismatch', 'Voided', 'NotTrenovaDocument', 'SentFromTrenova', 'ApplyFailed')),
    CONSTRAINT "ck_accounting_inbound_changes_amount" CHECK ("amount_minor" >= 0),
    CONSTRAINT "ck_accounting_inbound_changes_decided" CHECK ("status" NOT IN ('Applied', 'Ignored') OR "decided_at" IS NOT NULL),
    CONSTRAINT "ck_accounting_inbound_changes_proposed" CHECK ("status" <> 'Proposed' OR "reason" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_inbound_changes_external" ON "accounting_inbound_changes" ("organization_id", "business_unit_id", "connection_id", "kind", "external_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_inbound_changes_status" ON "accounting_inbound_changes" ("organization_id", "business_unit_id", "connection_id", "status", "detected_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_inbound_changes_detected" ON "accounting_inbound_changes" ("detected_at")WHERE "status" = 'Detected';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_sync_records_external" ON "accounting_sync_records" ("organization_id", "business_unit_id", "connection_id", "external_id")WHERE "external_id" IS NOT NULL;
