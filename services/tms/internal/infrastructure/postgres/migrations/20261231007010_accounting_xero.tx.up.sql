-- Xero is the second accounting system. Connections and app credentials accept
-- it, account references carry a provider-neutral class the mapping scorer and
-- ledger mode read, and a connection keeps the organisation's short code for
-- links into the accounting system.
ALTER TABLE "accounting_connections"
    DROP CONSTRAINT IF EXISTS "ck_accounting_connections_integration_type";

--bun:split
ALTER TABLE "accounting_connections"
    ADD CONSTRAINT "ck_accounting_connections_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero'));

--bun:split
ALTER TABLE "accounting_connections"
    ADD COLUMN IF NOT EXISTS "external_short_code" varchar(20);

--bun:split
ALTER TABLE "accounting_app_credentials"
    DROP CONSTRAINT IF EXISTS "ck_accounting_app_credentials_integration_type";

--bun:split
ALTER TABLE "accounting_app_credentials"
    ADD CONSTRAINT "ck_accounting_app_credentials_integration_type" CHECK ("integration_type" IN ('QuickBooksOnline', 'Xero'));

--bun:split
ALTER TABLE "accounting_reference_objects"
    ADD COLUMN IF NOT EXISTS "account_class" varchar(30);

--bun:split
UPDATE "accounting_reference_objects"
SET "account_class" = CASE
    WHEN "account_sub_type" = 'UndepositedFunds' THEN 'UndepositedFunds'
    WHEN "account_type" = 'Accounts Receivable' THEN 'Receivable'
    WHEN "account_type" = 'Accounts Payable' THEN 'Payable'
    WHEN "account_type" = 'Bank' THEN 'Bank'
    WHEN "account_type" = 'Income' THEN 'Income'
    WHEN "account_type" = 'Other Income' THEN 'OtherIncome'
    WHEN "account_type" = 'Cost of Goods Sold' THEN 'CostOfSales'
    WHEN "account_type" = 'Expense' THEN 'Expense'
    WHEN "account_type" = 'Other Expense' THEN 'OtherExpense'
    WHEN "account_type" IN ('Other Current Asset', 'Fixed Asset', 'Other Asset') THEN 'Asset'
    WHEN "account_type" IN ('Credit Card', 'Other Current Liability', 'Long Term Liability') THEN 'Liability'
    WHEN "account_type" = 'Equity' THEN 'Equity'
    ELSE NULL
END
WHERE "kind" = 'Account' AND "account_class" IS NULL;

--bun:split
ALTER TABLE "accounting_reference_objects"
    ADD CONSTRAINT "ck_accounting_reference_objects_account_class" CHECK ("account_class" IS NULL OR "account_class" IN ('Receivable', 'Payable', 'Bank', 'UndepositedFunds', 'Income', 'OtherIncome', 'CostOfSales', 'Expense', 'OtherExpense', 'Asset', 'Liability', 'Equity'));
