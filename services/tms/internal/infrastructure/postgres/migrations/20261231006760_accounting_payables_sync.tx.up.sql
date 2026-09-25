-- When the connection began sending owner-operator driver settlements. Null
-- means driver settlements are not sent, which is the default (decision D5).
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "driver_settlements_enabled_at" bigint;

--bun:split
-- The payable account a driver settlement's journal credited, stamped at
-- posting the way carrier settlements stamp posted_ap_account_id, so a bill
-- names the same account the GL does after the control changes.
ALTER TABLE "driver_settlements"
    ADD COLUMN IF NOT EXISTS "posted_payable_account_id" varchar(100);

--bun:split
ALTER TABLE "driver_settlements"
    DROP CONSTRAINT IF EXISTS "fk_driver_settlements_posted_payable_account";

--bun:split
ALTER TABLE "driver_settlements"
    ADD CONSTRAINT "fk_driver_settlements_posted_payable_account" FOREIGN KEY ("posted_payable_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE RESTRICT;

--bun:split
-- Payables join the outbox: vendors, bills and bill payments for carrier and
-- driver settlements.
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_object_type";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_object_type" CHECK ("object_type" IN ('Customer', 'Invoice', 'CreditMemo', 'DebitMemo', 'CustomerPayment', 'CreditApplication', 'CarrierVendor', 'DriverVendor', 'CarrierBill', 'CarrierBillPayment', 'DriverBill', 'DriverBillPayment'));

--bun:split
ALTER TABLE "accounting_sync_records"
    DROP CONSTRAINT IF EXISTS "ck_accounting_sync_records_source_event";

--bun:split
ALTER TABLE "accounting_sync_records"
    ADD CONSTRAINT "ck_accounting_sync_records_source_event" CHECK ("source_event" IN ('InvoicePosted', 'CreditMemoPosted', 'DebitMemoPosted', 'AdjustmentCreditMemo', 'CustomerPaymentPosted', 'CustomerPaymentApplied', 'CustomerPaymentReversed', 'CreditMemoApplied', 'CreditMemoUnapplied', 'CustomerUpdated', 'CarrierSettlementPosted', 'CarrierSettlementVoided', 'CarrierSettlementPaid', 'DriverSettlementPosted', 'DriverSettlementVoided', 'DriverSettlementPaid', 'CarrierUpdated', 'DriverUpdated', 'DependencyOf', 'SafetyNet', 'Backfill'));

--bun:split
-- Owner-operator drivers map to vendors, and Trenova GL accounts map to
-- provider accounts for the lines of a bill.
ALTER TABLE "accounting_mappings"
    DROP CONSTRAINT IF EXISTS "ck_accounting_mappings_target_type";

--bun:split
ALTER TABLE "accounting_mappings"
    ADD CONSTRAINT "ck_accounting_mappings_target_type" CHECK ("target_type" IN ('AccountRole', 'LineType', 'AccessorialCharge', 'ItemRole', 'Customer', 'Carrier', 'Driver', 'GLAccount', 'PaymentTerm', 'PaymentMethod'));
