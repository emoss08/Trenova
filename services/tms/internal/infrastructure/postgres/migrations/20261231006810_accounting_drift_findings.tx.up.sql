-- When the connection was last compared with the provider, and how it went.
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "drift_checked_at" bigint,
    ADD COLUMN IF NOT EXISTS "drift_error_category" varchar(30),
    ADD COLUMN IF NOT EXISTS "drift_error_message" text;

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_drift_error_category";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_drift_error_category" CHECK ("drift_error_category" IS NULL OR "drift_error_category" IN ('Transient', 'RateLimited', 'Auth', 'Validation', 'Mapping', 'ClosedPeriod', 'Currency', 'Duplicate', 'NotFound', 'Conflict', 'Configuration'));

--bun:split
-- A drift fix pushes Trenova's value as an update, or creates a document again
-- when the provider deleted or voided it.
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_operation";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_operation" CHECK ("operation" IN ('Create', 'Update', 'Void', 'Recreate'));

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_source_event";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'CarrierSettlementPosted', 'CarrierSettlementVoided', 'CarrierSettlementPaid', 'DriverSettlementPosted', 'DriverSettlementVoided', 'DriverSettlementPaid', 'CarrierUpdated', 'DriverUpdated', 'DependencyOf', 'SafetyNet', 'Backfill', 'DriftResolved'));

--bun:split
-- Where a document Trenova sent and the provider's copy of it differ now: one
-- open finding per document and kind, with both values, until a fix or a later
-- compare resolves it, or a person dismisses it.
CREATE TABLE IF NOT EXISTS "accounting_drift_findings"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "connection_id" varchar(100) NOT NULL,
    "object_type" varchar(30) NOT NULL,
    "object_id" varchar(100) NOT NULL,
    "object_number" varchar(100),
    "party_id" varchar(100),
    "party_name" varchar(200),
    "external_id" varchar(100),
    "external_url" text,
    "kind" varchar(30) NOT NULL,
    "currency_code" varchar(3) NOT NULL,
    "trenova_minor" bigint,
    "provider_minor" bigint,
    "difference_minor" bigint,
    "trenova_state" varchar(50),
    "provider_state" varchar(50),
    "detail" jsonb NOT NULL DEFAULT '[]'::jsonb,
    "provider_modified_at" bigint,
    "provider_modified_by" varchar(200),
    "status" varchar(20) NOT NULL DEFAULT 'Open',
    "resolution" varchar(30),
    "resolution_note" text,
    "fix_object_type" varchar(40),
    "fix_object_id" varchar(100),
    "resolved_by_id" varchar(100),
    "resolved_at" bigint,
    "detected_at" bigint NOT NULL,
    "last_seen_at" bigint NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_accounting_drift_findings_open" ON "accounting_drift_findings"("organization_id", "business_unit_id", "connection_id", "object_type", "object_id", "kind")
    WHERE "status" = 'Open';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_drift_findings_status" ON "accounting_drift_findings"("organization_id", "business_unit_id", "connection_id", "status", "detected_at");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_drift_findings_object" ON "accounting_drift_findings"("organization_id", "business_unit_id", "object_id");
