DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'account_type_enum') THEN
            CREATE TYPE account_type_enum AS ENUM ('Asset', 'Liability', 'Equity', 'Revenue', 'Expense');
        END IF;
    END
$$;

--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'cash_flow_type_enum') THEN
            CREATE TYPE cash_flow_type_enum AS ENUM ('Operating', 'Investing', 'Financing');
        END IF;
    END
$$;

--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'account_sub_type_enum') THEN
            CREATE TYPE account_sub_type_enum AS ENUM (
                'CurrentAsset',
                'FixedAsset',
                'OtherAsset',
                'CurrentLiability',
                'LongTermLiability',
                'Equity',
                'Revenue',
                'CostOfGoodsSold',
                'Expense',
                'OtherIncome',
                'OtherExpense');
        END IF;
    END
$$;

--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'account_classification_type_enum') THEN
            CREATE TYPE account_classification_type_enum AS ENUM (
                'Bank',
                'Cash',
                'AccountsReceivable',
                'AccountPayable',
                'Inventory',
                'PrepaidExpenses',
                'AccruedExpenses',
                'OtherCurrentAsset',
                'FixedAsset');
        END IF;
    END
$$;

--bun:split

CREATE TABLE
    IF NOT EXISTS "general_ledger_accounts"
(
    "id"               uuid              NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid              NOT NULL,
    "organization_id"  uuid              NOT NULL,
    "status"           status_enum       NOT NULL DEFAULT 'Active',
    "account_number"   VARCHAR(7)        NOT NULL,
    "account_type"     account_type_enum NOT NULL,
    "cash_flow_type"   cash_flow_type_enum,
    "account_sub_type" account_sub_type_enum,
    "account_class"    account_classification_type_enum,
    "balance"          NUMERIC(14, 2)    NOT NULL DEFAULT 0,
    "interest_rate"    NUMERIC(5, 2),
    "notes"            TEXT,
    "is_tax_relevant"  BOOLEAN           NOT NULL DEFAULT FALSE,
    "is_reconciled"    BOOLEAN           NOT NULL DEFAULT FALSE,
    "version"          BIGINT            NOT NULL,
    "created_at"       TIMESTAMPTZ       NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ       NOT NULL DEFAULT current_timestamp,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "gl_account_account_number_organization_id_unq" ON "general_ledger_accounts" (LOWER("account_number"), organization_id);
CREATE INDEX idx_general_ledger_accounts_account_number ON general_ledger_accounts (account_number);
CREATE INDEX idx_general_ledger_accounts_org_bu ON general_ledger_accounts (organization_id, business_unit_id);
CREATE INDEX idx_general_ledger_accounts_created_at ON general_ledger_accounts (created_at);

--bun:split

COMMENT ON COLUMN general_ledger_accounts.id IS 'Unique identifier for the general ledger account, generated as a UUID';
COMMENT ON COLUMN general_ledger_accounts.business_unit_id IS 'Foreign key referencing the business unit that this general ledger account belongs to';
COMMENT ON COLUMN general_ledger_accounts.organization_id IS 'Foreign key referencing the organization that this general ledger account belongs to';
COMMENT ON COLUMN general_ledger_accounts.status IS 'The current status of the general ledger account, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN general_ledger_accounts.account_number IS 'A unique account number for identifying the general ledger account, limited to 7 characters';
COMMENT ON COLUMN general_ledger_accounts.account_type IS 'The type of account, using the account_type_enum (e.g., Asset, Liability, Equity, Revenue, Expense)';
COMMENT ON COLUMN general_ledger_accounts.cash_flow_type IS 'The cash flow type of the account, using the cash_flow_type_enum (e.g., Operating, Investing, Financing)';
COMMENT ON COLUMN general_ledger_accounts.account_sub_type IS 'The sub-type of the account, using the account_sub_type_enum (e.g., CurrentAsset, FixedAsset, OtherAsset, CurrentLiability, LongTermLiability, Equity, Revenue, CostOfGoodsSold, Expense, OtherIncome, OtherExpense)';
COMMENT ON COLUMN general_ledger_accounts.account_class IS 'The classification of the account, using the account_classification_type_enum (e.g., Bank, Cash, AccountsReceivable, AccountPayable, Inventory, PrepaidExpenses, AccruedExpenses, OtherCurrentAsset, FixedAsset)';
COMMENT ON COLUMN general_ledger_accounts.balance IS 'The current balance of the account';
COMMENT ON COLUMN general_ledger_accounts.interest_rate IS 'The interest rate of the account';
COMMENT ON COLUMN general_ledger_accounts.notes IS 'Additional notes or comments about the account';
COMMENT ON COLUMN general_ledger_accounts.is_tax_relevant IS 'Flag indicating if the account is tax relevant';
COMMENT ON COLUMN general_ledger_accounts.is_reconciled IS 'Flag indicating if the account is reconciled';
COMMENT ON COLUMN general_ledger_accounts.created_at IS 'Timestamp of when the general ledger account was created, defaults to the current timestamp';
COMMENT ON COLUMN general_ledger_accounts.updated_at IS 'Timestamp of the last update to the general ledger account, defaults to the current timestamp';
