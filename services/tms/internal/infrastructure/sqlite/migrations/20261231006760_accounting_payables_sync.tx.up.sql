-- Hand-written: SQLite cannot alter a CHECK constraint, so accounting_sync_records
-- and accounting_mappings are rebuilt with the widened object type, source event
-- and target type checks, and their rows copied across, as 20261231006680 did.
-- accounting_sync_attempts references accounting_sync_records with ON DELETE
-- CASCADE, so its rows are set aside and cleared while the old table still
-- stands, then put back once the rebuilt table holds every record again.
-- Source: 20261231006760_accounting_payables_sync.tx.up.sql

ALTER TABLE "accounting_connections" ADD COLUMN "driver_settlements_enabled_at" INTEGER;

--bun:split

ALTER TABLE "driver_settlements" ADD COLUMN "posted_payable_account_id" TEXT;

--bun:split

CREATE TEMP TABLE "accounting_sync_attempts_backup" AS SELECT * FROM "accounting_sync_attempts";

--bun:split

CREATE TABLE "accounting_sync_records_rebuilt"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "connection_id" TEXT NOT NULL,
    "object_type" TEXT NOT NULL,
    "object_id" TEXT NOT NULL,
    "object_number" TEXT,
    "operation" TEXT NOT NULL,
    "source_event" TEXT NOT NULL,
    "idempotency_key" TEXT NOT NULL,
    "request_id" TEXT NOT NULL,
    "revision" INTEGER NOT NULL DEFAULT 1,
    "document_date" INTEGER,
    "depends_on_record_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Queued',
    "attempt_count" INTEGER NOT NULL DEFAULT 0,
    "next_attempt_at" INTEGER,
    "lease_expires_at" INTEGER,
    "external_id" TEXT,
    "external_doc_number" TEXT,
    "external_url" TEXT,
    "external_refs" TEXT NOT NULL DEFAULT '{}',
    "payload_hash" TEXT,
    "payload" TEXT,
    "mapping_ids" TEXT NOT NULL DEFAULT '[]',
    "error_category" TEXT,
    "error_code" TEXT,
    "error_message" TEXT,
    "resolution" TEXT,
    "queued_at" INTEGER NOT NULL,
    "started_at" INTEGER,
    "synced_at" INTEGER,
    "released_by_id" TEXT,
    "skipped_by_id" TEXT,
    "skipped_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_sync_records" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_accounting_sync_records_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_connection" FOREIGN KEY ("connection_id", "business_unit_id", "organization_id") REFERENCES "accounting_connections"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_sync_records_released_by" FOREIGN KEY ("released_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_accounting_sync_records_skipped_by" FOREIGN KEY ("skipped_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_accounting_sync_records_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication', 'CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment')),
    CONSTRAINT "ck_accounting_sync_records_operation" CHECK ("operation" IN ('Create', 'Update', 'Void')),
    CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'CarrierSettlementPosted', 'CarrierSettlementVoided', 'CarrierSettlementPaid', 'DriverSettlementPosted', 'DriverSettlementVoided', 'DriverSettlementPaid', 'CarrierUpdated', 'DriverUpdated', 'DependencyOf', 'SafetyNet', 'Backfill')),
    CONSTRAINT "ck_accounting_sync_records_status" CHECK ("status" IN ('Queued', 'AwaitingApproval', 'InFlight', 'Retrying', 'Synced', 'Blocked', 'DeadLettered', 'Skipped', 'Superseded')),
    CONSTRAINT "ck_accounting_sync_records_error_category" CHECK ("error_category" IS NULL OR "error_category" IN ('Transient', 'RateLimited', 'Auth', 'Validation', 'Mapping', 'ClosedPeriod', 'Currency', 'Duplicate', 'NotFound', 'Conflict', 'Configuration')),
    CONSTRAINT "ck_accounting_sync_records_revision" CHECK ("revision" >= 1),
    CONSTRAINT "ck_accounting_sync_records_attempts" CHECK ("attempt_count" >= 0),
    CONSTRAINT "ck_accounting_sync_records_synced" CHECK ("status" <> 'Synced' OR ("synced_at" IS NOT NULL)),
    CONSTRAINT "ck_accounting_sync_records_skipped" CHECK ("status" <> 'Skipped' OR "skipped_reason" IS NOT NULL)
);

--bun:split

INSERT INTO "accounting_sync_records_rebuilt" ("id", "business_unit_id", "organization_id", "connection_id", "object_type", "object_id", "object_number", "operation", "source_event", "idempotency_key", "request_id", "revision", "document_date", "depends_on_record_id", "status", "attempt_count", "next_attempt_at", "lease_expires_at", "external_id", "external_doc_number", "external_url", "external_refs", "payload_hash", "payload", "mapping_ids", "error_category", "error_code", "error_message", "resolution", "queued_at", "started_at", "synced_at", "released_by_id", "skipped_by_id", "skipped_reason", "version", "created_at", "updated_at")
SELECT "id", "business_unit_id", "organization_id", "connection_id", "object_type", "object_id", "object_number", "operation", "source_event", "idempotency_key", "request_id", "revision", "document_date", "depends_on_record_id", "status", "attempt_count", "next_attempt_at", "lease_expires_at", "external_id", "external_doc_number", "external_url", "external_refs", "payload_hash", "payload", "mapping_ids", "error_category", "error_code", "error_message", "resolution", "queued_at", "started_at", "synced_at", "released_by_id", "skipped_by_id", "skipped_reason", "version", "created_at", "updated_at"
FROM "accounting_sync_records";

--bun:split

DELETE FROM "accounting_sync_attempts";

--bun:split

DROP TABLE "accounting_sync_records";

--bun:split

ALTER TABLE "accounting_sync_records_rebuilt" RENAME TO "accounting_sync_records";

--bun:split

CREATE UNIQUE INDEX "uq_accounting_sync_records_idempotency" ON "accounting_sync_records" ("organization_id", "business_unit_id", "connection_id", "idempotency_key");

--bun:split

CREATE INDEX "idx_accounting_sync_records_due" ON "accounting_sync_records" ("organization_id", "business_unit_id", "connection_id", "status", "next_attempt_at");

--bun:split

CREATE INDEX "idx_accounting_sync_records_dispatch" ON "accounting_sync_records" ("next_attempt_at", "lease_expires_at")WHERE "status" IN ('Queued', 'Retrying', 'InFlight');

--bun:split

CREATE INDEX "idx_accounting_sync_records_object" ON "accounting_sync_records" ("organization_id", "business_unit_id", "object_type", "object_id");

--bun:split

CREATE INDEX "idx_accounting_sync_records_queued" ON "accounting_sync_records" ("organization_id", "business_unit_id", "connection_id", "queued_at" DESC);

--bun:split

INSERT INTO "accounting_sync_attempts" SELECT * FROM "accounting_sync_attempts_backup";

--bun:split

DROP TABLE "accounting_sync_attempts_backup";

--bun:split

CREATE TABLE "accounting_mappings_rebuilt"(
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
    CONSTRAINT "ck_accounting_mappings_target_type" CHECK ("target_type" IN ('AccountRole', 'LineType', 'AccessorialCharge', 'ItemRole', 'Customer', 'Carrier', 'Driver', 'GLAccount', 'PaymentTerm', 'PaymentMethod')),
    CONSTRAINT "ck_accounting_mappings_provider_kind" CHECK ("provider_kind" IN ('Account', 'Item', 'Customer', 'Vendor', 'Term', 'PaymentMethod')),
    CONSTRAINT "ck_accounting_mappings_state" CHECK ("state" IN ('Unmatched', 'Proposed', 'Confirmed')),
    CONSTRAINT "ck_accounting_mappings_source" CHECK ("source" IS NULL OR "source" IN ('Suggested', 'Model', 'Manual', 'CreatedInProvider', 'Agent')),
    CONSTRAINT "ck_accounting_mappings_target_identity" CHECK (("trenova_object_id" IS NULL) <> ("trenova_key" IS NULL)),
    CONSTRAINT "ck_accounting_mappings_external_when_matched" CHECK (("state" = 'Unmatched') = ("external_id" IS NULL)),
    CONSTRAINT "ck_accounting_mappings_confidence" CHECK ("confidence" IS NULL OR ("confidence" >= 0 AND "confidence" <= 1))
);

--bun:split

INSERT INTO "accounting_mappings_rebuilt" ("id", "business_unit_id", "organization_id", "connection_id", "target_type", "trenova_object_id", "trenova_key", "target_label", "search_label", "provider_kind", "external_id", "external_name", "state", "source", "confidence", "reason", "signals", "confirmed_by_id", "confirmed_at", "version", "created_at", "updated_at")
SELECT "id", "business_unit_id", "organization_id", "connection_id", "target_type", "trenova_object_id", "trenova_key", "target_label", "search_label", "provider_kind", "external_id", "external_name", "state", "source", "confidence", "reason", "signals", "confirmed_by_id", "confirmed_at", "version", "created_at", "updated_at"
FROM "accounting_mappings";

--bun:split

DROP TABLE "accounting_mappings";

--bun:split

ALTER TABLE "accounting_mappings_rebuilt" RENAME TO "accounting_mappings";

--bun:split

CREATE UNIQUE INDEX "uq_accounting_mappings_target" ON "accounting_mappings" ("organization_id", "business_unit_id", "connection_id", "target_type", COALESCE("trenova_object_id", ''), COALESCE("trenova_key", ''));

--bun:split

CREATE INDEX "idx_accounting_mappings_review" ON "accounting_mappings" ("organization_id", "business_unit_id", "connection_id", "target_type", "state");

--bun:split

CREATE INDEX "idx_accounting_mappings_external" ON "accounting_mappings" ("organization_id", "business_unit_id", "connection_id", "provider_kind", "external_id")WHERE "external_id" IS NOT NULL;
