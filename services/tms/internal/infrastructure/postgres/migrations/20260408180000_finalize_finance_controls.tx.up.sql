DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_type
        WHERE typname = 'payment_term_enum'
    ) THEN
        ALTER TYPE payment_term_enum ADD VALUE IF NOT EXISTS 'Net10';
    END IF;
END;
$$;

--bun:split
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'accounting_basis_enum') THEN
        CREATE TYPE accounting_basis_enum AS ENUM ('Accrual', 'Cash');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'revenue_recognition_policy_enum') THEN
        CREATE TYPE revenue_recognition_policy_enum AS ENUM ('OnInvoicePost', 'OnCashReceipt');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'expense_recognition_policy_enum') THEN
        CREATE TYPE expense_recognition_policy_enum AS ENUM ('OnVendorBillPost', 'OnCashDisbursement');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'journal_posting_mode_enum') THEN
        CREATE TYPE journal_posting_mode_enum AS ENUM ('Manual', 'Automatic');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'journal_source_event_enum') THEN
        CREATE TYPE journal_source_event_enum AS ENUM (
            'InvoicePosted',
            'CreditMemoPosted',
            'DebitMemoPosted',
            'CustomerPaymentPosted',
            'VendorBillPosted',
            'VendorPaymentPosted'
        );
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'manual_journal_entry_policy_enum') THEN
        CREATE TYPE manual_journal_entry_policy_enum AS ENUM ('AllowAll', 'AdjustmentOnly', 'Disallow');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'journal_reversal_policy_enum') THEN
        CREATE TYPE journal_reversal_policy_enum AS ENUM ('Disallow', 'NextOpenPeriod');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'period_close_mode_enum') THEN
        CREATE TYPE period_close_mode_enum AS ENUM ('ManualOnly', 'SystemScheduled');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'locked_period_posting_policy_enum') THEN
        CREATE TYPE locked_period_posting_policy_enum AS ENUM ('BlockSubledgerAllowManualJe');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'closed_period_posting_policy_enum') THEN
        CREATE TYPE closed_period_posting_policy_enum AS ENUM ('RequireReopen', 'PostToNextOpen');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'reconciliation_mode_enum') THEN
        CREATE TYPE reconciliation_mode_enum AS ENUM ('Disabled', 'WarnOnly', 'BlockPosting');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'currency_mode_enum') THEN
        CREATE TYPE currency_mode_enum AS ENUM ('SingleCurrency', 'MultiCurrency');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'exchange_rate_date_policy_enum') THEN
        CREATE TYPE exchange_rate_date_policy_enum AS ENUM ('DocumentDate', 'AccountingDate');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'exchange_rate_override_policy_enum') THEN
        CREATE TYPE exchange_rate_override_policy_enum AS ENUM ('Allow', 'RequireApproval', 'Disallow');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'enforcement_level_enum') THEN
        CREATE TYPE enforcement_level_enum AS ENUM ('Ignore', 'Warn', 'RequireReview', 'Block');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_exception_disposition_enum') THEN
        CREATE TYPE billing_exception_disposition_enum AS ENUM ('RouteToBillingReview', 'ReturnToOperations');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'ready_to_bill_assignment_mode_enum') THEN
        CREATE TYPE ready_to_bill_assignment_mode_enum AS ENUM ('ManualOnly', 'AutomaticWhenEligible');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_queue_transfer_mode_enum') THEN
        CREATE TYPE billing_queue_transfer_mode_enum AS ENUM ('ManualOnly', 'AutomaticWhenReady');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'invoice_draft_creation_mode_enum') THEN
        CREATE TYPE invoice_draft_creation_mode_enum AS ENUM ('ManualOnly', 'AutomaticWhenTransferred');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'invoice_posting_mode_enum') THEN
        CREATE TYPE invoice_posting_mode_enum AS ENUM ('ManualReviewRequired', 'AutomaticWhenNoBlockingExceptions');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'rate_variance_auto_resolution_mode_enum') THEN
        CREATE TYPE rate_variance_auto_resolution_mode_enum AS ENUM ('Disabled', 'BypassReviewWithinTolerance');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adjustment_eligibility_policy_enum') THEN
        CREATE TYPE adjustment_eligibility_policy_enum AS ENUM ('Disallow', 'AllowWithApproval', 'AllowWithoutApproval');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adjustment_accounting_date_policy_enum') THEN
        CREATE TYPE adjustment_accounting_date_policy_enum AS ENUM ('UseOriginalIfOpenElseNextOpen', 'AlwaysNextOpen');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'closed_period_adjustment_policy_enum') THEN
        CREATE TYPE closed_period_adjustment_policy_enum AS ENUM ('Disallow', 'RequireReopen', 'PostInNextOpenPeriodWithApproval');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'requirement_policy_enum') THEN
        CREATE TYPE requirement_policy_enum AS ENUM ('Optional', 'Required');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'adjustment_attachment_policy_enum') THEN
        CREATE TYPE adjustment_attachment_policy_enum AS ENUM ('Optional', 'RequiredForCreditOrWriteOff', 'RequiredForAll');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'approval_policy_enum') THEN
        CREATE TYPE approval_policy_enum AS ENUM ('None', 'Always', 'AmountThreshold');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'write_off_approval_policy_enum') THEN
        CREATE TYPE write_off_approval_policy_enum AS ENUM ('Disallow', 'AlwaysRequireApproval', 'RequireApprovalAboveThreshold');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'replacement_invoice_review_policy_enum') THEN
        CREATE TYPE replacement_invoice_review_policy_enum AS ENUM ('NoAdditionalReview', 'RequireReviewWhenEconomicTermsChange', 'AlwaysRequireReview');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'customer_credit_balance_policy_enum') THEN
        CREATE TYPE customer_credit_balance_policy_enum AS ENUM ('Disallow', 'AllowUnappliedCredit');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'over_credit_policy_enum') THEN
        CREATE TYPE over_credit_policy_enum AS ENUM ('Block', 'AllowWithApproval');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'superseded_invoice_visibility_policy_enum') THEN
        CREATE TYPE superseded_invoice_visibility_policy_enum AS ENUM ('ShowCurrentOnlyExternally', 'ShowCurrentAndSupersededExternally');
    END IF;
END;
$$;

--bun:split
ALTER TABLE accounting_controls
    ADD COLUMN IF NOT EXISTS accounting_basis accounting_basis_enum NOT NULL DEFAULT 'Accrual',
    ADD COLUMN IF NOT EXISTS revenue_recognition_policy revenue_recognition_policy_enum NOT NULL DEFAULT 'OnInvoicePost',
    ADD COLUMN IF NOT EXISTS expense_recognition_policy expense_recognition_policy_enum NOT NULL DEFAULT 'OnVendorBillPost',
    ADD COLUMN IF NOT EXISTS journal_posting_mode journal_posting_mode_enum NOT NULL DEFAULT 'Manual',
    ADD COLUMN IF NOT EXISTS auto_post_source_events journal_source_event_enum[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS manual_journal_entry_policy manual_journal_entry_policy_enum NOT NULL DEFAULT 'AdjustmentOnly',
    ADD COLUMN IF NOT EXISTS require_manual_je_approval BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS journal_reversal_policy journal_reversal_policy_enum NOT NULL DEFAULT 'NextOpenPeriod',
    ADD COLUMN IF NOT EXISTS period_close_mode period_close_mode_enum NOT NULL DEFAULT 'ManualOnly',
    ADD COLUMN IF NOT EXISTS require_period_close_approval BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS locked_period_posting_policy locked_period_posting_policy_enum NOT NULL DEFAULT 'BlockSubledgerAllowManualJe',
    ADD COLUMN IF NOT EXISTS closed_period_posting_policy closed_period_posting_policy_enum NOT NULL DEFAULT 'RequireReopen',
    ADD COLUMN IF NOT EXISTS require_reconciliation_to_close BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS reconciliation_mode reconciliation_mode_enum NOT NULL DEFAULT 'Disabled',
    ADD COLUMN IF NOT EXISTS reconciliation_tolerance_amount NUMERIC(19, 4) NOT NULL DEFAULT 0.0000,
    ADD COLUMN IF NOT EXISTS notify_on_reconciliation_exception BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS currency_mode currency_mode_enum NOT NULL DEFAULT 'SingleCurrency',
    ADD COLUMN IF NOT EXISTS functional_currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS exchange_rate_date_policy exchange_rate_date_policy_enum NOT NULL DEFAULT 'DocumentDate',
    ADD COLUMN IF NOT EXISTS exchange_rate_override_policy exchange_rate_override_policy_enum NOT NULL DEFAULT 'RequireApproval',
    ADD COLUMN IF NOT EXISTS default_tax_liability_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_write_off_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS realized_fx_gain_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS realized_fx_loss_account_id VARCHAR(100);

--bun:split
UPDATE accounting_controls
SET
    accounting_basis = CASE
        WHEN accounting_method = 'Cash' THEN 'Cash'::accounting_basis_enum
        ELSE 'Accrual'::accounting_basis_enum
    END,
    revenue_recognition_policy = CASE
        WHEN accounting_method = 'Cash' OR revenue_recognition_method = 'OnPayment' THEN 'OnCashReceipt'::revenue_recognition_policy_enum
        ELSE 'OnInvoicePost'::revenue_recognition_policy_enum
    END,
    expense_recognition_policy = CASE
        WHEN accounting_method = 'Cash' OR expense_recognition_method = 'OnPayment' THEN 'OnCashDisbursement'::expense_recognition_policy_enum
        ELSE 'OnVendorBillPost'::expense_recognition_policy_enum
    END,
    journal_posting_mode = CASE
        WHEN auto_create_journal_entries THEN 'Automatic'::journal_posting_mode_enum
        ELSE 'Manual'::journal_posting_mode_enum
    END,
    auto_post_source_events = CASE
        WHEN auto_create_journal_entries THEN ARRAY_REMOVE(ARRAY[
            CASE
                WHEN accounting_method = 'Cash' OR revenue_recognition_method = 'OnPayment'
                    THEN 'CustomerPaymentPosted'::journal_source_event_enum
                ELSE 'InvoicePosted'::journal_source_event_enum
            END,
            CASE
                WHEN accounting_method = 'Cash' OR expense_recognition_method = 'OnPayment'
                    THEN 'VendorPaymentPosted'::journal_source_event_enum
                ELSE 'VendorBillPosted'::journal_source_event_enum
            END
        ], NULL)
        ELSE ARRAY[]::journal_source_event_enum[]
    END,
    manual_journal_entry_policy = CASE
        WHEN restrict_manual_journal_entries THEN 'Disallow'::manual_journal_entry_policy_enum
        ELSE 'AdjustmentOnly'::manual_journal_entry_policy_enum
    END,
    require_manual_je_approval = CASE
        WHEN restrict_manual_journal_entries THEN FALSE
        ELSE require_journal_entry_approval
    END,
    journal_reversal_policy = CASE
        WHEN enable_journal_entry_reversal THEN 'NextOpenPeriod'::journal_reversal_policy_enum
        ELSE 'Disallow'::journal_reversal_policy_enum
    END,
    period_close_mode = CASE
        WHEN auto_close_periods THEN 'SystemScheduled'::period_close_mode_enum
        ELSE 'ManualOnly'::period_close_mode_enum
    END,
    require_period_close_approval = CASE
        WHEN auto_close_periods THEN FALSE
        ELSE require_period_end_approval
    END,
    locked_period_posting_policy = 'BlockSubledgerAllowManualJe'::locked_period_posting_policy_enum,
    closed_period_posting_policy = CASE
        WHEN allow_posting_to_closed_periods THEN 'PostToNextOpen'::closed_period_posting_policy_enum
        ELSE 'RequireReopen'::closed_period_posting_policy_enum
    END,
    require_reconciliation_to_close = halt_on_pending_reconciliation,
    reconciliation_mode = CASE
        WHEN NOT enable_reconciliation THEN 'Disabled'::reconciliation_mode_enum
        WHEN reconciliation_threshold_action = 'Block' THEN 'BlockPosting'::reconciliation_mode_enum
        ELSE 'WarnOnly'::reconciliation_mode_enum
    END,
    reconciliation_tolerance_amount = CASE
        WHEN enable_reconciliation THEN reconciliation_threshold
        ELSE 0.0000
    END,
    notify_on_reconciliation_exception = enable_reconciliation_notifications,
    currency_mode = CASE
        WHEN enable_multi_currency THEN 'MultiCurrency'::currency_mode_enum
        ELSE 'SingleCurrency'::currency_mode_enum
    END,
    functional_currency_code = UPPER(default_currency_code),
    exchange_rate_date_policy = 'DocumentDate'::exchange_rate_date_policy_enum,
    exchange_rate_override_policy = CASE
        WHEN enable_multi_currency THEN 'RequireApproval'::exchange_rate_override_policy_enum
        ELSE 'Disallow'::exchange_rate_override_policy_enum
    END,
    default_tax_liability_account_id = default_tax_account_id,
    default_write_off_account_id = COALESCE(default_expense_account_id, default_ar_account_id),
    realized_fx_gain_account_id = currency_gain_account_id,
    realized_fx_loss_account_id = currency_loss_account_id;

--bun:split
DROP TRIGGER IF EXISTS accounting_controls_validate_journal_entry_criteria_trigger ON accounting_controls;
DROP FUNCTION IF EXISTS accounting_controls_validate_journal_entry_criteria();

ALTER TABLE accounting_controls
    DROP CONSTRAINT IF EXISTS fk_accounting_controls_default_tax_account,
    DROP CONSTRAINT IF EXISTS fk_accounting_controls_default_deferred_revenue_account,
    DROP CONSTRAINT IF EXISTS fk_accounting_controls_default_cost_of_service_account,
    DROP CONSTRAINT IF EXISTS fk_accounting_controls_currency_gain_account,
    DROP CONSTRAINT IF EXISTS fk_accounting_controls_currency_loss_account;

ALTER TABLE accounting_controls
    ADD CONSTRAINT fk_accounting_controls_default_tax_liability_account
        FOREIGN KEY (default_tax_liability_account_id, organization_id, business_unit_id)
        REFERENCES gl_accounts (id, organization_id, business_unit_id)
        ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT fk_accounting_controls_default_write_off_account
        FOREIGN KEY (default_write_off_account_id, organization_id, business_unit_id)
        REFERENCES gl_accounts (id, organization_id, business_unit_id)
        ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT fk_accounting_controls_realized_fx_gain_account
        FOREIGN KEY (realized_fx_gain_account_id, organization_id, business_unit_id)
        REFERENCES gl_accounts (id, organization_id, business_unit_id)
        ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT fk_accounting_controls_realized_fx_loss_account
        FOREIGN KEY (realized_fx_loss_account_id, organization_id, business_unit_id)
        REFERENCES gl_accounts (id, organization_id, business_unit_id)
        ON UPDATE NO ACTION ON DELETE RESTRICT;

ALTER TABLE accounting_controls
    DROP COLUMN IF EXISTS accounting_method,
    DROP COLUMN IF EXISTS default_tax_account_id,
    DROP COLUMN IF EXISTS default_deferred_revenue_account_id,
    DROP COLUMN IF EXISTS default_cost_of_service_account_id,
    DROP COLUMN IF EXISTS auto_create_journal_entries,
    DROP COLUMN IF EXISTS journal_entry_criteria,
    DROP COLUMN IF EXISTS restrict_manual_journal_entries,
    DROP COLUMN IF EXISTS require_journal_entry_approval,
    DROP COLUMN IF EXISTS enable_journal_entry_reversal,
    DROP COLUMN IF EXISTS allow_posting_to_closed_periods,
    DROP COLUMN IF EXISTS require_period_end_approval,
    DROP COLUMN IF EXISTS auto_close_periods,
    DROP COLUMN IF EXISTS enable_reconciliation,
    DROP COLUMN IF EXISTS reconciliation_threshold,
    DROP COLUMN IF EXISTS reconciliation_threshold_action,
    DROP COLUMN IF EXISTS halt_on_pending_reconciliation,
    DROP COLUMN IF EXISTS enable_reconciliation_notifications,
    DROP COLUMN IF EXISTS revenue_recognition_method,
    DROP COLUMN IF EXISTS defer_revenue_until_paid,
    DROP COLUMN IF EXISTS expense_recognition_method,
    DROP COLUMN IF EXISTS accrue_expenses,
    DROP COLUMN IF EXISTS enable_automatic_tax_calculation,
    DROP COLUMN IF EXISTS require_document_attachment,
    DROP COLUMN IF EXISTS retain_deleted_entries,
    DROP COLUMN IF EXISTS enable_multi_currency,
    DROP COLUMN IF EXISTS default_currency_code,
    DROP COLUMN IF EXISTS currency_gain_account_id,
    DROP COLUMN IF EXISTS currency_loss_account_id;

--bun:split
ALTER TABLE billing_controls
    ADD COLUMN IF NOT EXISTS default_payment_term payment_term_enum NOT NULL DEFAULT 'Net30',
    ADD COLUMN IF NOT EXISTS default_invoice_terms TEXT,
    ADD COLUMN IF NOT EXISTS default_invoice_footer TEXT,
    ADD COLUMN IF NOT EXISTS show_due_date_on_invoice BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS show_balance_due_on_invoice BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS ready_to_bill_assignment_mode ready_to_bill_assignment_mode_enum NOT NULL DEFAULT 'ManualOnly',
    ADD COLUMN IF NOT EXISTS billing_queue_transfer_mode billing_queue_transfer_mode_enum NOT NULL DEFAULT 'ManualOnly',
    ADD COLUMN IF NOT EXISTS billing_queue_transfer_schedule transfer_schedule_enum,
    ADD COLUMN IF NOT EXISTS billing_queue_transfer_batch_size INTEGER,
    ADD COLUMN IF NOT EXISTS invoice_draft_creation_mode invoice_draft_creation_mode_enum NOT NULL DEFAULT 'ManualOnly',
    ADD COLUMN IF NOT EXISTS invoice_posting_mode invoice_posting_mode_enum NOT NULL DEFAULT 'ManualReviewRequired',
    ADD COLUMN IF NOT EXISTS auto_invoice_batch_size INTEGER,
    ADD COLUMN IF NOT EXISTS notify_on_auto_invoice_creation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS shipment_billing_requirement_enforcement enforcement_level_enum NOT NULL DEFAULT 'Block',
    ADD COLUMN IF NOT EXISTS rate_validation_enforcement enforcement_level_enum NOT NULL DEFAULT 'RequireReview',
    ADD COLUMN IF NOT EXISTS billing_exception_disposition billing_exception_disposition_enum NOT NULL DEFAULT 'RouteToBillingReview',
    ADD COLUMN IF NOT EXISTS notify_on_billing_exceptions BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS rate_variance_tolerance_percent NUMERIC(9, 6) NOT NULL DEFAULT 0.000000,
    ADD COLUMN IF NOT EXISTS rate_variance_auto_resolution_mode rate_variance_auto_resolution_mode_enum NOT NULL DEFAULT 'Disabled';

--bun:split
UPDATE billing_controls
SET
    default_payment_term = payment_term,
    default_invoice_terms = invoice_terms,
    default_invoice_footer = invoice_footer,
    show_due_date_on_invoice = show_invoice_due_date,
    show_balance_due_on_invoice = show_amount_due,
    ready_to_bill_assignment_mode = CASE
        WHEN auto_mark_ready_to_bill THEN 'AutomaticWhenEligible'::ready_to_bill_assignment_mode_enum
        ELSE 'ManualOnly'::ready_to_bill_assignment_mode_enum
    END,
    billing_queue_transfer_mode = CASE
        WHEN auto_transfer THEN 'AutomaticWhenReady'::billing_queue_transfer_mode_enum
        ELSE 'ManualOnly'::billing_queue_transfer_mode_enum
    END,
    billing_queue_transfer_schedule = CASE
        WHEN auto_transfer THEN transfer_schedule
        ELSE NULL
    END,
    billing_queue_transfer_batch_size = CASE
        WHEN auto_transfer THEN transfer_batch_size
        ELSE NULL
    END,
    invoice_draft_creation_mode = CASE
        WHEN auto_bill THEN 'AutomaticWhenTransferred'::invoice_draft_creation_mode_enum
        ELSE 'ManualOnly'::invoice_draft_creation_mode_enum
    END,
    invoice_posting_mode = CASE
        WHEN auto_bill THEN 'AutomaticWhenNoBlockingExceptions'::invoice_posting_mode_enum
        ELSE 'ManualReviewRequired'::invoice_posting_mode_enum
    END,
    auto_invoice_batch_size = CASE
        WHEN auto_bill THEN auto_bill_batch_size
        ELSE NULL
    END,
    notify_on_auto_invoice_creation = CASE
        WHEN auto_bill THEN send_auto_bill_notifications
        ELSE FALSE
    END,
    shipment_billing_requirement_enforcement = CASE
        WHEN enforce_customer_billing_req THEN 'Block'::enforcement_level_enum
        ELSE 'Ignore'::enforcement_level_enum
    END,
    rate_validation_enforcement = CASE
        WHEN validate_customer_rates THEN 'RequireReview'::enforcement_level_enum
        ELSE 'Ignore'::enforcement_level_enum
    END,
    billing_exception_disposition = CASE
        WHEN billing_exception_handling = 'Reject' THEN 'ReturnToOperations'::billing_exception_disposition_enum
        ELSE 'RouteToBillingReview'::billing_exception_disposition_enum
    END,
    notify_on_billing_exceptions = TRUE,
    rate_variance_tolerance_percent = 0.000000,
    rate_variance_auto_resolution_mode = 'Disabled'::rate_variance_auto_resolution_mode_enum;

--bun:split
ALTER TABLE billing_controls
    DROP COLUMN IF EXISTS payment_term,
    DROP COLUMN IF EXISTS show_invoice_due_date,
    DROP COLUMN IF EXISTS invoice_terms,
    DROP COLUMN IF EXISTS invoice_footer,
    DROP COLUMN IF EXISTS show_amount_due,
    DROP COLUMN IF EXISTS auto_transfer,
    DROP COLUMN IF EXISTS transfer_schedule,
    DROP COLUMN IF EXISTS transfer_batch_size,
    DROP COLUMN IF EXISTS auto_mark_ready_to_bill,
    DROP COLUMN IF EXISTS enforce_customer_billing_req,
    DROP COLUMN IF EXISTS validate_customer_rates,
    DROP COLUMN IF EXISTS auto_bill,
    DROP COLUMN IF EXISTS send_auto_bill_notifications,
    DROP COLUMN IF EXISTS auto_bill_batch_size,
    DROP COLUMN IF EXISTS billing_exception_handling,
    DROP COLUMN IF EXISTS rate_discrepancy_threshold,
    DROP COLUMN IF EXISTS auto_resolve_minor_discrepancies,
    DROP COLUMN IF EXISTS allow_invoice_consolidation,
    DROP COLUMN IF EXISTS consolidation_period_days,
    DROP COLUMN IF EXISTS group_consolidated_invoices;

--bun:split
CREATE TABLE IF NOT EXISTS invoice_adjustment_controls (
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    partially_paid_invoice_adjustment_policy adjustment_eligibility_policy_enum NOT NULL DEFAULT 'AllowWithApproval',
    paid_invoice_adjustment_policy adjustment_eligibility_policy_enum NOT NULL DEFAULT 'Disallow',
    disputed_invoice_adjustment_policy adjustment_eligibility_policy_enum NOT NULL DEFAULT 'AllowWithApproval',
    adjustment_accounting_date_policy adjustment_accounting_date_policy_enum NOT NULL DEFAULT 'UseOriginalIfOpenElseNextOpen',
    closed_period_adjustment_policy closed_period_adjustment_policy_enum NOT NULL DEFAULT 'PostInNextOpenPeriodWithApproval',
    adjustment_reason_requirement requirement_policy_enum NOT NULL DEFAULT 'Required',
    adjustment_attachment_requirement adjustment_attachment_policy_enum NOT NULL DEFAULT 'RequiredForAll',
    standard_adjustment_approval_policy approval_policy_enum NOT NULL DEFAULT 'AmountThreshold',
    standard_adjustment_approval_threshold NUMERIC(19, 4),
    write_off_approval_policy write_off_approval_policy_enum NOT NULL DEFAULT 'RequireApprovalAboveThreshold',
    write_off_approval_threshold NUMERIC(19, 4),
    rerate_variance_tolerance_percent NUMERIC(9, 6) NOT NULL DEFAULT 0.000000,
    replacement_invoice_review_policy replacement_invoice_review_policy_enum NOT NULL DEFAULT 'RequireReviewWhenEconomicTermsChange',
    customer_credit_balance_policy customer_credit_balance_policy_enum NOT NULL DEFAULT 'AllowUnappliedCredit',
    over_credit_policy over_credit_policy_enum NOT NULL DEFAULT 'Block',
    superseded_invoice_visibility_policy superseded_invoice_visibility_policy_enum NOT NULL DEFAULT 'ShowCurrentOnlyExternally',
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    updated_at BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT pk_invoice_adjustment_controls PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_adjustment_controls_business_unit FOREIGN KEY (business_unit_id) REFERENCES business_units (id) ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT fk_invoice_adjustment_controls_organization FOREIGN KEY (organization_id) REFERENCES organizations (id) ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT uq_invoice_adjustment_controls_organization UNIQUE (organization_id)
);

--bun:split
INSERT INTO invoice_adjustment_controls (
    id,
    business_unit_id,
    organization_id
)
SELECT
    CONCAT('iac_', replace(gen_random_uuid()::text, '-', '')) AS id,
    seed.business_unit_id,
    seed.organization_id
FROM (
    SELECT DISTINCT ON (organization_id)
        organization_id,
        business_unit_id
    FROM (
        SELECT organization_id, business_unit_id FROM billing_controls
        UNION ALL
        SELECT organization_id, business_unit_id FROM accounting_controls
    ) sources
    ORDER BY organization_id, business_unit_id
) seed
WHERE NOT EXISTS (
    SELECT 1
    FROM invoice_adjustment_controls iac
    WHERE iac.organization_id = seed.organization_id
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_controls_tenant
    ON invoice_adjustment_controls (business_unit_id, organization_id);

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_controls_created_at
    ON invoice_adjustment_controls (created_at);

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_controls_updated_at
    ON invoice_adjustment_controls (updated_at);

--bun:split
CREATE OR REPLACE FUNCTION invoice_adjustment_controls_update_timestamp()
RETURNS TRIGGER
AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS invoice_adjustment_controls_update_timestamp_trigger ON invoice_adjustment_controls;

CREATE TRIGGER invoice_adjustment_controls_update_timestamp_trigger
    BEFORE UPDATE ON invoice_adjustment_controls
    FOR EACH ROW
    EXECUTE FUNCTION invoice_adjustment_controls_update_timestamp();

--bun:split
DROP TYPE IF EXISTS accounting_method_enum;
DROP TYPE IF EXISTS journal_entry_criteria_enum;
DROP TYPE IF EXISTS reconciliation_threshold_action_enum;
DROP TYPE IF EXISTS revenue_recognition_enum;
DROP TYPE IF EXISTS expense_recognition_enum;
DROP TYPE IF EXISTS billing_exception_handling_enum;
DROP TYPE IF EXISTS approval_requirement_enum;
