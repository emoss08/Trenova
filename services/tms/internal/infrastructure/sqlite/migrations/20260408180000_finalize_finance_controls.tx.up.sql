-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260408180000_finalize_finance_controls.tx.up.sql

ALTER TABLE "accounting_controls" ADD COLUMN "accounting_basis" TEXT NOT NULL DEFAULT 'Accrual';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "revenue_recognition_policy" TEXT NOT NULL DEFAULT 'OnInvoicePost';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "expense_recognition_policy" TEXT NOT NULL DEFAULT 'OnVendorBillPost';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "journal_posting_mode" TEXT NOT NULL DEFAULT 'Manual';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "auto_post_source_events" TEXT NOT NULL DEFAULT '[]';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "manual_journal_entry_policy" TEXT NOT NULL DEFAULT 'AdjustmentOnly';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "require_manual_je_approval" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "journal_reversal_policy" TEXT NOT NULL DEFAULT 'NextOpenPeriod';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "period_close_mode" TEXT NOT NULL DEFAULT 'ManualOnly';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "require_period_close_approval" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "locked_period_posting_policy" TEXT NOT NULL DEFAULT 'BlockSubledgerAllowManualJe';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "closed_period_posting_policy" TEXT NOT NULL DEFAULT 'RequireReopen';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "require_reconciliation_to_close" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "reconciliation_mode" TEXT NOT NULL DEFAULT 'Disabled';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "reconciliation_tolerance_amount" REAL NOT NULL DEFAULT 0.0000;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "notify_on_reconciliation_exception" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "currency_mode" TEXT NOT NULL DEFAULT 'SingleCurrency';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "functional_currency_code" TEXT NOT NULL DEFAULT 'USD';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "exchange_rate_date_policy" TEXT NOT NULL DEFAULT 'DocumentDate';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "exchange_rate_override_policy" TEXT NOT NULL DEFAULT 'RequireApproval';

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_tax_liability_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_write_off_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "realized_fx_gain_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "realized_fx_loss_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "accounting_method";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "auto_create_journal_entries";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "journal_entry_criteria";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "restrict_manual_journal_entries";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "require_journal_entry_approval";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "enable_journal_entry_reversal";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "allow_posting_to_closed_periods";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "require_period_end_approval";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "auto_close_periods";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "enable_reconciliation";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "reconciliation_threshold";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "reconciliation_threshold_action";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "halt_on_pending_reconciliation";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "enable_reconciliation_notifications";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "revenue_recognition_method";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "defer_revenue_until_paid";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "expense_recognition_method";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "accrue_expenses";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "enable_automatic_tax_calculation";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "require_document_attachment";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "retain_deleted_entries";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "enable_multi_currency";

--bun:split

ALTER TABLE "accounting_controls" DROP COLUMN "default_currency_code";

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "default_payment_term" TEXT NOT NULL DEFAULT 'Net30';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "default_invoice_terms" TEXT;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "default_invoice_footer" TEXT;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "show_due_date_on_invoice" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "show_balance_due_on_invoice" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "ready_to_bill_assignment_mode" TEXT NOT NULL DEFAULT 'ManualOnly';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "billing_queue_transfer_mode" TEXT NOT NULL DEFAULT 'ManualOnly';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "billing_queue_transfer_schedule" TEXT;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "billing_queue_transfer_batch_size" INTEGER;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "invoice_draft_creation_mode" TEXT NOT NULL DEFAULT 'ManualOnly';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "invoice_posting_mode" TEXT NOT NULL DEFAULT 'ManualReviewRequired';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "auto_invoice_batch_size" INTEGER;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "notify_on_auto_invoice_creation" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "shipment_billing_requirement_enforcement" TEXT NOT NULL DEFAULT 'Block';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "rate_validation_enforcement" TEXT NOT NULL DEFAULT 'RequireReview';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "billing_exception_disposition" TEXT NOT NULL DEFAULT 'RouteToBillingReview';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "notify_on_billing_exceptions" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "rate_variance_tolerance_percent" REAL NOT NULL DEFAULT 0.000000;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "rate_variance_auto_resolution_mode" TEXT NOT NULL DEFAULT 'Disabled';

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "payment_term";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "show_invoice_due_date";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "invoice_terms";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "invoice_footer";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "show_amount_due";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "auto_transfer";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "transfer_schedule";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "auto_mark_ready_to_bill";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "enforce_customer_billing_req";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "validate_customer_rates";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "auto_bill";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "send_auto_bill_notifications";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "billing_exception_handling";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "auto_resolve_minor_discrepancies";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "allow_invoice_consolidation";

--bun:split

ALTER TABLE "billing_controls" DROP COLUMN "group_consolidated_invoices";

--bun:split

CREATE TABLE IF NOT EXISTS invoice_adjustment_controls (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "partially_paid_invoice_adjustment_policy" TEXT NOT NULL DEFAULT 'AllowWithApproval',
    "paid_invoice_adjustment_policy" TEXT NOT NULL DEFAULT 'Disallow',
    "disputed_invoice_adjustment_policy" TEXT NOT NULL DEFAULT 'AllowWithApproval',
    "adjustment_accounting_date_policy" TEXT NOT NULL DEFAULT 'UseOriginalIfOpenElseNextOpen',
    "closed_period_adjustment_policy" TEXT NOT NULL DEFAULT 'PostInNextOpenPeriodWithApproval',
    "adjustment_reason_requirement" TEXT NOT NULL DEFAULT 'Required',
    "adjustment_attachment_requirement" TEXT NOT NULL DEFAULT 'RequiredForAll',
    "standard_adjustment_approval_policy" TEXT NOT NULL DEFAULT 'AmountThreshold',
    "standard_adjustment_approval_threshold" REAL,
    "write_off_approval_policy" TEXT NOT NULL DEFAULT 'RequireApprovalAboveThreshold',
    "write_off_approval_threshold" REAL,
    "rerate_variance_tolerance_percent" REAL NOT NULL DEFAULT 0.000000,
    "replacement_invoice_review_policy" TEXT NOT NULL DEFAULT 'RequireReviewWhenEconomicTermsChange',
    "customer_credit_balance_policy" TEXT NOT NULL DEFAULT 'AllowUnappliedCredit',
    "over_credit_policy" TEXT NOT NULL DEFAULT 'Block',
    "superseded_invoice_visibility_policy" TEXT NOT NULL DEFAULT 'ShowCurrentOnlyExternally',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_adjustment_controls PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_adjustment_controls_business_unit FOREIGN KEY (business_unit_id) REFERENCES business_units (id) ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT fk_invoice_adjustment_controls_organization FOREIGN KEY (organization_id) REFERENCES organizations (id) ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT uq_invoice_adjustment_controls_organization UNIQUE (organization_id)
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_controls_tenant
    ON invoice_adjustment_controls (business_unit_id, organization_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_controls_created_at
    ON invoice_adjustment_controls (created_at);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_controls_updated_at
    ON invoice_adjustment_controls (updated_at);
