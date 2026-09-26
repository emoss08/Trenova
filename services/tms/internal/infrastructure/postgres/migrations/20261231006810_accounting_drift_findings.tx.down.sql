DROP TABLE IF EXISTS "accounting_drift_findings";

--bun:split
UPDATE "accounting_sync_records" SET "source_event" = 'DependencyOf' WHERE "source_event" = 'DriftResolved';

--bun:split
UPDATE "accounting_sync_records" SET "operation" = 'Create' WHERE "operation" = 'Recreate';

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_source_event";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'CarrierSettlementPosted', 'CarrierSettlementVoided', 'CarrierSettlementPaid', 'DriverSettlementPosted', 'DriverSettlementVoided', 'DriverSettlementPaid', 'CarrierUpdated', 'DriverUpdated', 'DependencyOf', 'SafetyNet', 'Backfill'));

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_operation";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_operation" CHECK ("operation" IN ('Create', 'Update', 'Void'));

--bun:split
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_drift_error_category";

--bun:split
ALTER TABLE "accounting_connections"
    DROP COLUMN IF EXISTS "drift_error_message",
    DROP COLUMN IF EXISTS "drift_error_category",
    DROP COLUMN IF EXISTS "drift_checked_at";
