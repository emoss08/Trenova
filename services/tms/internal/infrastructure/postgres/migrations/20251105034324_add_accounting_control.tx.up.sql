CREATE TYPE "accounting_method_enum" AS ENUM(
    'Accrual',
    'Cash',
    'Hybrid'
);

CREATE TYPE "journal_entry_criteria_enum" AS ENUM(
    'InvoicePosted',
    'BillPosted',
    'PaymentReceived',
    'PaymentMade',
    'DeliveryComplete',
    'ShipmentDispatched'
);

CREATE TYPE "reconciliation_threshold_action_enum" AS ENUM(
    'Warn',
    'Block',
    'Notify'
);

CREATE TYPE "revenue_recognition_enum" AS ENUM(
    'OnDelivery',
    'OnBilling',
    'OnPayment',
    'OnPickup'
);

CREATE TYPE "expense_recognition_enum" AS ENUM(
    'OnIncurrence',
    'OnAccrual',
    'OnPayment'
);

--bun:split
CREATE TABLE IF NOT EXISTS "accounting_controls"(
    -- Identity
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Accounting Method
    "accounting_method" accounting_method_enum NOT NULL DEFAULT 'Accrual',
    -- Default GL Accounts
    "default_revenue_account_id" varchar(100),
    "default_expense_account_id" varchar(100),
    "default_ar_account_id" varchar(100),
    "default_ap_account_id" varchar(100),
    "default_tax_account_id" varchar(100),
    "default_deferred_revenue_account_id" varchar(100),
    "default_cost_of_service_account_id" varchar(100),
    "default_retained_earnings_account_id" varchar(100),
    -- Journal Entry Automation
    "auto_create_journal_entries" boolean NOT NULL DEFAULT FALSE,
    "journal_entry_criteria" jsonb NOT NULL DEFAULT '[]' ::jsonb,
    -- Journal Entry Controls
    "restrict_manual_journal_entries" boolean NOT NULL DEFAULT FALSE,
    "require_journal_entry_approval" boolean NOT NULL DEFAULT TRUE,
    "enable_journal_entry_reversal" boolean NOT NULL DEFAULT TRUE,
    -- Period Controls
    "allow_posting_to_closed_periods" boolean NOT NULL DEFAULT FALSE,
    "require_period_end_approval" boolean NOT NULL DEFAULT TRUE,
    "auto_close_periods" boolean NOT NULL DEFAULT FALSE,
    -- Reconciliation Settings
    "enable_reconciliation" boolean NOT NULL DEFAULT FALSE,
    "reconciliation_threshold" numeric(19, 4) NOT NULL DEFAULT 0.0050,
    "reconciliation_threshold_action" reconciliation_threshold_action_enum NOT NULL DEFAULT 'Warn',
    "halt_on_pending_reconciliation" boolean NOT NULL DEFAULT FALSE,
    "enable_reconciliation_notifications" boolean NOT NULL DEFAULT TRUE,
    -- Revenue Recognition
    "revenue_recognition_method" revenue_recognition_enum NOT NULL DEFAULT 'OnDelivery',
    "defer_revenue_until_paid" boolean NOT NULL DEFAULT FALSE,
    -- Expense Recognition
    "expense_recognition_method" expense_recognition_enum NOT NULL DEFAULT 'OnIncurrence',
    "accrue_expenses" boolean NOT NULL DEFAULT TRUE,
    -- Tax Settings
    "enable_automatic_tax_calculation" boolean NOT NULL DEFAULT FALSE,
    -- Audit & Compliance
    "require_document_attachment" boolean NOT NULL DEFAULT FALSE,
    "retain_deleted_entries" boolean NOT NULL DEFAULT TRUE,
    -- Multi-Currency
    "enable_multi_currency" boolean NOT NULL DEFAULT FALSE,
    "default_currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "currency_gain_account_id" varchar(100),
    "currency_loss_account_id" varchar(100),
    -- Versioning & Timestamps
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Primary Key
    CONSTRAINT "pk_accounting_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    -- Tenant FKs
    CONSTRAINT "fk_accounting_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- GL Account FKs (all composite, all RESTRICT on delete to prevent orphaned references)
    CONSTRAINT "fk_accounting_controls_default_revenue_account" FOREIGN KEY ("default_revenue_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_expense_account" FOREIGN KEY ("default_expense_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_ar_account" FOREIGN KEY ("default_ar_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_ap_account" FOREIGN KEY ("default_ap_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_tax_account" FOREIGN KEY ("default_tax_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_deferred_revenue_account" FOREIGN KEY ("default_deferred_revenue_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_cost_of_service_account" FOREIGN KEY ("default_cost_of_service_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_retained_earnings_account" FOREIGN KEY ("default_retained_earnings_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_currency_gain_account" FOREIGN KEY ("currency_gain_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_currency_loss_account" FOREIGN KEY ("currency_loss_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    -- One accounting control per organization
    CONSTRAINT "uq_accounting_controls_organization" UNIQUE ("organization_id"),
    -- Reconciliation threshold must be non-negative
    CONSTRAINT "ck_accounting_controls_reconciliation_threshold_positive" CHECK ("reconciliation_threshold" >= 0),
    -- Currency code must be exactly 3 characters
    CONSTRAINT "ck_accounting_controls_currency_code_length" CHECK (length("default_currency_code") = 3),
    -- Journal entry criteria must be a JSON array (element validation done via trigger)
    CONSTRAINT "ck_accounting_controls_journal_entry_criteria_is_array" CHECK (jsonb_typeof("journal_entry_criteria") = 'array'),
    -- Accounting method coherence: cash basis requires OnPayment for revenue
    CONSTRAINT "ck_accounting_controls_cash_revenue_coherence" CHECK ("accounting_method" != 'Cash' OR "revenue_recognition_method" = 'OnPayment'),
    -- Accounting method coherence: cash/hybrid basis requires OnPayment for expenses
    CONSTRAINT "ck_accounting_controls_cash_expense_coherence" CHECK ("accounting_method" NOT IN ('Cash', 'Hybrid') OR "expense_recognition_method" = 'OnPayment'),
    -- Flag coherence: defer revenue not applicable under cash basis
    CONSTRAINT "ck_accounting_controls_defer_revenue_coherence" CHECK ("accounting_method" != 'Cash' OR "defer_revenue_until_paid" = FALSE),
    -- Flag coherence: accrue expenses not applicable under cash or hybrid
    CONSTRAINT "ck_accounting_controls_accrue_expenses_coherence" CHECK ("accounting_method" NOT IN ('Cash', 'Hybrid') OR "accrue_expenses" = FALSE)
);

--bun:split
-- Validates that every element in journal_entry_criteria is a known value.
-- Postgres does not allow subqueries in CHECK constraints, so this runs as a trigger.
CREATE OR REPLACE FUNCTION accounting_controls_validate_journal_entry_criteria()
    RETURNS TRIGGER
    AS $$
DECLARE
    _elem text;
    _valid_values text[] := ARRAY['InvoicePosted', 'BillPosted', 'PaymentReceived', 'PaymentMade', 'DeliveryComplete', 'ShipmentDispatched'];
BEGIN
    FOR _elem IN
    SELECT
        jsonb_array_elements_text(NEW.journal_entry_criteria)
        LOOP
            IF _elem != ALL (_valid_values) THEN
                RAISE EXCEPTION 'Invalid journal entry criteria value: "%". Valid values: %', _elem, array_to_string(_valid_values, ', ')
                    USING ERRCODE = 'check_violation';
                END IF;
            END LOOP;
            RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS accounting_controls_validate_journal_entry_criteria_trigger ON accounting_controls;

CREATE TRIGGER accounting_controls_validate_journal_entry_criteria_trigger
    BEFORE INSERT OR UPDATE OF journal_entry_criteria ON accounting_controls
    FOR EACH ROW
    EXECUTE FUNCTION accounting_controls_validate_journal_entry_criteria();

--bun:split
-- Tenant lookup (most common access pattern)
CREATE INDEX IF NOT EXISTS "idx_accounting_controls_tenant" ON "accounting_controls"("business_unit_id", "organization_id");

-- Timestamp indexes for audit queries and change tracking
CREATE INDEX IF NOT EXISTS "idx_accounting_controls_created_at" ON "accounting_controls"("created_at");

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_updated_at" ON "accounting_controls"("updated_at");

-- Partial indexes on GL account FKs (only index non-null values since these are optional)
CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_revenue_account" ON "accounting_controls"("default_revenue_account_id")
WHERE
    "default_revenue_account_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_expense_account" ON "accounting_controls"("default_expense_account_id")
WHERE
    "default_expense_account_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_ar_account" ON "accounting_controls"("default_ar_account_id")
WHERE
    "default_ar_account_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_ap_account" ON "accounting_controls"("default_ap_account_id")
WHERE
    "default_ap_account_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_deferred_revenue_account" ON "accounting_controls"("default_deferred_revenue_account_id")
WHERE
    "default_deferred_revenue_account_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_cost_of_service_account" ON "accounting_controls"("default_cost_of_service_account_id")
WHERE
    "default_cost_of_service_account_id" IS NOT NULL;

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_retained_earnings_account" ON "accounting_controls"("default_retained_earnings_account_id")
WHERE
    "default_retained_earnings_account_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE accounting_controls IS 'Stores per-organization configuration for accounting controls, recognition methods, and validation rules';

COMMENT ON COLUMN accounting_controls.accounting_method IS 'Top-level accounting method (Accrual, Cash, Hybrid) that constrains valid recognition methods';

COMMENT ON COLUMN accounting_controls.journal_entry_criteria IS 'JSON array of journal entry trigger events (e.g. InvoicePosted, PaymentReceived)';

COMMENT ON COLUMN accounting_controls.reconciliation_threshold IS 'Maximum allowed variance in monetary reconciliation (NUMERIC(19,4) for sub-cent precision)';

COMMENT ON COLUMN accounting_controls.defer_revenue_until_paid IS 'When true, revenue is booked to a deferred revenue liability account until payment is received (not applicable under cash basis)';

COMMENT ON COLUMN accounting_controls.accrue_expenses IS 'When true, expenses are accrued when incurred rather than when paid (not applicable under cash or hybrid basis)';

--bun:split
CREATE OR REPLACE FUNCTION accounting_controls_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS accounting_controls_update_timestamp_trigger ON accounting_controls;

CREATE TRIGGER accounting_controls_update_timestamp_trigger
    BEFORE UPDATE ON accounting_controls
    FOR EACH ROW
    EXECUTE FUNCTION accounting_controls_update_timestamp();

--bun:split
ALTER TABLE accounting_controls
    ALTER COLUMN organization_id SET STATISTICS 1000;

ALTER TABLE accounting_controls
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

