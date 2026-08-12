-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20251105034324_add_accounting_control.tx.up.sql

CREATE TABLE IF NOT EXISTS "accounting_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "accounting_method" TEXT NOT NULL DEFAULT 'Accrual',
    "default_revenue_account_id" TEXT,
    "default_expense_account_id" TEXT,
    "default_ar_account_id" TEXT,
    "default_ap_account_id" TEXT,
    "default_tax_account_id" TEXT,
    "default_deferred_revenue_account_id" TEXT,
    "default_cost_of_service_account_id" TEXT,
    "default_retained_earnings_account_id" TEXT,
    "auto_create_journal_entries" INTEGER NOT NULL DEFAULT 0,
    "journal_entry_criteria" TEXT NOT NULL DEFAULT '[]',
    "restrict_manual_journal_entries" INTEGER NOT NULL DEFAULT 0,
    "require_journal_entry_approval" INTEGER NOT NULL DEFAULT 1,
    "enable_journal_entry_reversal" INTEGER NOT NULL DEFAULT 1,
    "allow_posting_to_closed_periods" INTEGER NOT NULL DEFAULT 0,
    "require_period_end_approval" INTEGER NOT NULL DEFAULT 1,
    "auto_close_periods" INTEGER NOT NULL DEFAULT 0,
    "enable_reconciliation" INTEGER NOT NULL DEFAULT 0,
    "reconciliation_threshold" NUMERIC NOT NULL DEFAULT 0.0050,
    "reconciliation_threshold_action" TEXT NOT NULL DEFAULT 'Warn',
    "halt_on_pending_reconciliation" INTEGER NOT NULL DEFAULT 0,
    "enable_reconciliation_notifications" INTEGER NOT NULL DEFAULT 1,
    "revenue_recognition_method" TEXT NOT NULL DEFAULT 'OnDelivery',
    "defer_revenue_until_paid" INTEGER NOT NULL DEFAULT 0,
    "expense_recognition_method" TEXT NOT NULL DEFAULT 'OnIncurrence',
    "accrue_expenses" INTEGER NOT NULL DEFAULT 1,
    "enable_automatic_tax_calculation" INTEGER NOT NULL DEFAULT 0,
    "require_document_attachment" INTEGER NOT NULL DEFAULT 0,
    "retain_deleted_entries" INTEGER NOT NULL DEFAULT 1,
    "enable_multi_currency" INTEGER NOT NULL DEFAULT 0,
    "default_currency_code" TEXT NOT NULL DEFAULT 'USD',
    "currency_gain_account_id" TEXT,
    "currency_loss_account_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_accounting_controls" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_accounting_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_accounting_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
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
    CONSTRAINT "uq_accounting_controls_organization" UNIQUE ("organization_id"),
    CONSTRAINT "ck_accounting_controls_reconciliation_threshold_positive" CHECK ("reconciliation_threshold" >= 0),
    CONSTRAINT "ck_accounting_controls_currency_code_length" CHECK (length("default_currency_code") = 3),
    CONSTRAINT "ck_accounting_controls_cash_revenue_coherence" CHECK ("accounting_method" != 'Cash' OR "revenue_recognition_method" = 'OnPayment'),
    CONSTRAINT "ck_accounting_controls_cash_expense_coherence" CHECK ("accounting_method" NOT IN ('Cash', 'Hybrid') OR "expense_recognition_method" = 'OnPayment'),
    CONSTRAINT "ck_accounting_controls_defer_revenue_coherence" CHECK ("accounting_method" != 'Cash' OR "defer_revenue_until_paid" = FALSE),
    CONSTRAINT "ck_accounting_controls_accrue_expenses_coherence" CHECK ("accounting_method" NOT IN ('Cash', 'Hybrid') OR "accrue_expenses" = FALSE)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_tenant" ON "accounting_controls" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_created_at" ON "accounting_controls" ("created_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_updated_at" ON "accounting_controls" ("updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_revenue_account" ON "accounting_controls" ("default_revenue_account_id")WHERE
    "default_revenue_account_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_expense_account" ON "accounting_controls" ("default_expense_account_id")WHERE
    "default_expense_account_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_ar_account" ON "accounting_controls" ("default_ar_account_id")WHERE
    "default_ar_account_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_ap_account" ON "accounting_controls" ("default_ap_account_id")WHERE
    "default_ap_account_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_deferred_revenue_account" ON "accounting_controls" ("default_deferred_revenue_account_id")WHERE
    "default_deferred_revenue_account_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_cost_of_service_account" ON "accounting_controls" ("default_cost_of_service_account_id")WHERE
    "default_cost_of_service_account_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_accounting_controls_default_retained_earnings_account" ON "accounting_controls" ("default_retained_earnings_account_id")WHERE
    "default_retained_earnings_account_id" IS NOT NULL;
