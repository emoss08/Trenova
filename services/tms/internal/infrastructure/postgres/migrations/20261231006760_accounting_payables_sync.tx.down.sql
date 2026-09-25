DELETE FROM "accounting_mappings" WHERE "target_type" IN ('Driver', 'GLAccount');

--bun:split
ALTER TABLE "accounting_mappings"
    DROP CONSTRAINT IF EXISTS "ck_accounting_mappings_target_type";

--bun:split
ALTER TABLE "accounting_mappings"
    ADD CONSTRAINT "ck_accounting_mappings_target_type" CHECK ("target_type" IN ('AccountRole', 'LineType', 'AccessorialCharge', 'ItemRole', 'Customer', 'Carrier', 'PaymentTerm', 'PaymentMethod'));

--bun:split
DELETE FROM "accounting_sync_records" WHERE "object_type" IN ('CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment');

--bun:split
UPDATE "accounting_sync_records" SET "source_event" = 'DependencyOf' WHERE "source_event" IN ('CarrierSettlementPosted', 'CarrierSettlementVoided', 'CarrierSettlementPaid', 'DriverSettlementPosted', 'DriverSettlementVoided', 'DriverSettlementPaid', 'CarrierUpdated', 'DriverUpdated');

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_source_event";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'DependencyOf', 'SafetyNet', 'Backfill'));

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_object_type";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication'));

--bun:split
ALTER TABLE "driver_settlements"
    DROP CONSTRAINT IF EXISTS "fk_driver_settlements_posted_payable_account";

--bun:split
ALTER TABLE "driver_settlements"
    DROP COLUMN IF EXISTS "posted_payable_account_id";

--bun:split
ALTER TABLE "accounting_connections"
    DROP COLUMN IF EXISTS "driver_settlements_enabled_at";
