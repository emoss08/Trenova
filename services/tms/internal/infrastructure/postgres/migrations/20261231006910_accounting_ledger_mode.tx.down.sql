DROP INDEX IF EXISTS "idx_journal_entries_posted_accounting_date";

--bun:split
DELETE FROM "accounting_drift_findings" WHERE "kind" = 'TrialBalanceMismatch' OR "object_type" = 'GLAccount';

--bun:split
ALTER TABLE "accounting_drift_findings"
    DROP CONSTRAINT IF EXISTS "ck_accounting_drift_findings_kind";

--bun:split
ALTER TABLE "accounting_drift_findings"
    ADD CONSTRAINT "ck_accounting_drift_findings_kind" CHECK ("kind" IN ('AmountMismatch', 'StatusMismatch', 'DeletedInProvider', 'VoidedInProvider', 'CustomerBalanceMismatch'));

--bun:split
ALTER TABLE "accounting_drift_findings"
    DROP CONSTRAINT IF EXISTS "ck_accounting_drift_findings_object_type";

--bun:split
ALTER TABLE "accounting_drift_findings"
    ADD CONSTRAINT "ck_accounting_drift_findings_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication', 'CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment'));

--bun:split
DELETE FROM "accounting_sync_attempts" WHERE "sync_record_id" IN (
    SELECT "id" FROM "accounting_sync_records" WHERE "object_type" IN ('JournalEntry', 'JournalSummary')
);

--bun:split
DELETE FROM "accounting_sync_records" WHERE "object_type" IN ('JournalEntry', 'JournalSummary');

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_source_event";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'CarrierSettlementPosted', 'CarrierSettlementVoided', 'CarrierSettlementPaid', 'DriverSettlementPosted', 'DriverSettlementVoided', 'DriverSettlementPaid', 'CarrierUpdated', 'DriverUpdated', 'DependencyOf', 'SafetyNet', 'Backfill', 'DriftResolved'));

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_object_type";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication', 'CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment'));

--bun:split
UPDATE "accounting_connections" SET "setup_step" = 'Mappings' WHERE "setup_step" = 'Mode';

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_setup_step";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_setup_step" CHECK ("setup_step" IN ('Mappings', 'StartDate', 'Complete'));

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_fiscal_year_start_month",
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_ledger_granularity",
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_sync_mode",
    DROP COLUMN IF EXISTS "external_fiscal_year_start_month",
    DROP COLUMN IF EXISTS "ledger_opening_balances_sent_at",
    DROP COLUMN IF EXISTS "ledger_granularity",
    DROP COLUMN IF EXISTS "sync_mode";
