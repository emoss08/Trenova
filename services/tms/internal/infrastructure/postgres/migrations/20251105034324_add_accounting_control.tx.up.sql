CREATE TYPE "journal_entry_criteria_enum" AS ENUM(
    'ShipmentBilled',
    'PaymentReceived',
    'ExpenseRecognized',
    'DeliveryComplete'
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
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "auto_create_journal_entries" boolean NOT NULL DEFAULT FALSE,
    "journal_entry_criteria" journal_entry_criteria_enum NOT NULL DEFAULT 'ShipmentBilled',
    "default_revenue_account_id" varchar(100),
    "default_expense_account_id" varchar(100),
    "restrict_manual_journal_entries" boolean NOT NULL DEFAULT FALSE,
    "require_journal_entry_approval" boolean NOT NULL DEFAULT TRUE,
    "enable_journal_entry_reversal" boolean NOT NULL DEFAULT TRUE,
    "allow_posting_to_closed_periods" boolean NOT NULL DEFAULT FALSE,
    "require_period_end_approval" boolean NOT NULL DEFAULT TRUE,
    "auto_close_periods" boolean NOT NULL DEFAULT FALSE,
    "enable_reconciliation" boolean NOT NULL DEFAULT TRUE,
    "reconciliation_threshold" integer NOT NULL DEFAULT 50,
    "reconciliation_threshold_action" reconciliation_threshold_action_enum NOT NULL DEFAULT 'Warn',
    "halt_on_pending_reconciliation" boolean NOT NULL DEFAULT FALSE,
    "enable_reconciliation_notifications" boolean NOT NULL DEFAULT TRUE,
    "revenue_recognition_method" revenue_recognition_enum NOT NULL DEFAULT 'OnDelivery',
    "defer_revenue_until_paid" boolean NOT NULL DEFAULT FALSE,
    "expense_recognition_method" expense_recognition_enum NOT NULL DEFAULT 'OnIncurrence',
    "accrue_expenses" boolean NOT NULL DEFAULT TRUE,
    "enable_automatic_tax_calculation" boolean NOT NULL DEFAULT TRUE,
    "default_tax_account_id" varchar(100),
    "require_document_attachment" boolean NOT NULL DEFAULT FALSE,
    "retain_deleted_entries" boolean NOT NULL DEFAULT TRUE,
    "enable_multi_currency" boolean NOT NULL DEFAULT FALSE,
    "default_currency_code" varchar(3) NOT NULL DEFAULT 'USD',
    "currency_gain_account_id" varchar(100),
    "currency_loss_account_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_accounting_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_accounting_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_controls_default_revenue_account" FOREIGN KEY ("default_revenue_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_expense_account" FOREIGN KEY ("default_expense_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_default_tax_account" FOREIGN KEY ("default_tax_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_currency_gain_account" FOREIGN KEY ("currency_gain_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_accounting_controls_currency_loss_account" FOREIGN KEY ("currency_loss_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "uq_accounting_controls_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accounting_controls_business_unit" ON "accounting_controls"("business_unit_id", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_created_at" ON "accounting_controls"("created_at", "updated_at");

COMMENT ON TABLE accounting_controls IS 'Stores configuration for accounting controls and validation rules';

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

