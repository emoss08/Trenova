-- Ledger mode: a connection sends either documents or posted journal
-- entries, chosen before mappings and fixed once sync is enabled.
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "sync_mode" varchar(20) NOT NULL DEFAULT 'Document',
    ADD COLUMN IF NOT EXISTS "ledger_granularity" varchar(20),
    ADD COLUMN IF NOT EXISTS "ledger_opening_balances_sent_at" bigint,
    ADD COLUMN IF NOT EXISTS "external_fiscal_year_start_month" smallint NOT NULL DEFAULT 1;

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_sync_mode" CHECK ("sync_mode" IN ('Document', 'Ledger'));

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_ledger_granularity" CHECK (
        ("sync_mode" = 'Document' AND "ledger_granularity" IS NULL)
        OR ("sync_mode" = 'Ledger' AND "ledger_granularity" IN ('Detailed', 'DailySummary'))
    );

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_fiscal_year_start_month" CHECK ("external_fiscal_year_start_month" BETWEEN 1 AND 12);

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_setup_step";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_setup_step" CHECK ("setup_step" IN ('Mode', 'Mappings', 'StartDate', 'Complete'));

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_object_type";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication', 'CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment', 'JournalEntry', 'JournalSummary'));

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_source_event";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'CarrierSettlementPosted', 'CarrierSettlementVoided', 'CarrierSettlementPaid', 'DriverSettlementPosted', 'DriverSettlementVoided', 'DriverSettlementPaid', 'CarrierUpdated', 'DriverUpdated', 'DependencyOf', 'SafetyNet', 'Backfill', 'DriftResolved', 'JournalPosted', 'OpeningBalances'));

--bun:split
ALTER TABLE "accounting_drift_findings"
    DROP CONSTRAINT IF EXISTS "ck_accounting_drift_findings_object_type";

--bun:split
ALTER TABLE "accounting_drift_findings"
    ADD CONSTRAINT "ck_accounting_drift_findings_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication', 'CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment', 'JournalEntry', 'JournalSummary', 'GLAccount'));

--bun:split
ALTER TABLE "accounting_drift_findings"
    DROP CONSTRAINT IF EXISTS "ck_accounting_drift_findings_kind";

--bun:split
ALTER TABLE "accounting_drift_findings"
    ADD CONSTRAINT "ck_accounting_drift_findings_kind" CHECK ("kind" IN ('AmountMismatch', 'StatusMismatch', 'DeletedInProvider', 'VoidedInProvider', 'CustomerBalanceMismatch', 'TrialBalanceMismatch'));

--bun:split
-- Ledger mode reads posted entries by posting time and by accounting date.
CREATE INDEX IF NOT EXISTS "idx_journal_entries_posted_accounting_date"
    ON "journal_entries" ("organization_id", "business_unit_id", "accounting_date")
    WHERE "is_posted" = true;
