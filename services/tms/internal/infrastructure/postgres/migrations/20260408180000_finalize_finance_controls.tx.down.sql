DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'accounting_method_enum') THEN
        CREATE TYPE accounting_method_enum AS ENUM ('Accrual', 'Cash', 'Hybrid');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'journal_entry_criteria_enum') THEN
        CREATE TYPE journal_entry_criteria_enum AS ENUM (
            'InvoicePosted',
            'BillPosted',
            'PaymentReceived',
            'PaymentMade',
            'DeliveryComplete',
            'ShipmentDispatched'
        );
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'reconciliation_threshold_action_enum') THEN
        CREATE TYPE reconciliation_threshold_action_enum AS ENUM ('Warn', 'Block', 'Notify');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'revenue_recognition_enum') THEN
        CREATE TYPE revenue_recognition_enum AS ENUM ('OnDelivery', 'OnBilling', 'OnPayment', 'OnPickup');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'expense_recognition_enum') THEN
        CREATE TYPE expense_recognition_enum AS ENUM ('OnIncurrence', 'OnAccrual', 'OnPayment');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'billing_exception_handling_enum') THEN
        CREATE TYPE billing_exception_handling_enum AS ENUM ('Queue', 'Notify', 'AutoResolve', 'Reject');
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'approval_requirement_enum') THEN
        CREATE TYPE approval_requirement_enum AS ENUM ('None', 'Always', 'ThresholdBased', 'ExceptionOnly');
    END IF;
END;
$$;

--bun:split
ALTER TABLE accounting_controls
    ADD COLUMN IF NOT EXISTS accounting_method accounting_method_enum NOT NULL DEFAULT 'Accrual',
    ADD COLUMN IF NOT EXISTS default_tax_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_deferred_revenue_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_cost_of_service_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS auto_create_journal_entries BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS journal_entry_criteria JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS restrict_manual_journal_entries BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS require_journal_entry_approval BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS enable_journal_entry_reversal BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS allow_posting_to_closed_periods BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS require_period_end_approval BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS auto_close_periods BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS enable_reconciliation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS reconciliation_threshold NUMERIC(19, 4) NOT NULL DEFAULT 0.0050,
    ADD COLUMN IF NOT EXISTS reconciliation_threshold_action reconciliation_threshold_action_enum NOT NULL DEFAULT 'Warn',
    ADD COLUMN IF NOT EXISTS halt_on_pending_reconciliation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS enable_reconciliation_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS revenue_recognition_method revenue_recognition_enum NOT NULL DEFAULT 'OnBilling',
    ADD COLUMN IF NOT EXISTS defer_revenue_until_paid BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS expense_recognition_method expense_recognition_enum NOT NULL DEFAULT 'OnAccrual',
    ADD COLUMN IF NOT EXISTS accrue_expenses BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS enable_automatic_tax_calculation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS require_document_attachment BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS retain_deleted_entries BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS enable_multi_currency BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS default_currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS currency_gain_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS currency_loss_account_id VARCHAR(100);

--bun:split
UPDATE accounting_controls
SET
    accounting_method = CASE
        WHEN accounting_basis = 'Cash' THEN 'Cash'::accounting_method_enum
        ELSE 'Accrual'::accounting_method_enum
    END,
    default_tax_account_id = default_tax_liability_account_id,
    default_cost_of_service_account_id = default_expense_account_id,
    auto_create_journal_entries = (journal_posting_mode = 'Automatic'),
    journal_entry_criteria = '[]'::jsonb,
    restrict_manual_journal_entries = (manual_journal_entry_policy = 'Disallow'),
    require_journal_entry_approval = require_manual_je_approval,
    enable_journal_entry_reversal = (journal_reversal_policy = 'NextOpenPeriod'),
    allow_posting_to_closed_periods = (closed_period_posting_policy = 'PostToNextOpen'),
    require_period_end_approval = require_period_close_approval,
    auto_close_periods = (period_close_mode = 'SystemScheduled'),
    enable_reconciliation = (reconciliation_mode <> 'Disabled'),
    reconciliation_threshold = reconciliation_tolerance_amount,
    reconciliation_threshold_action = CASE
        WHEN reconciliation_mode = 'BlockPosting' THEN 'Block'::reconciliation_threshold_action_enum
        ELSE 'Warn'::reconciliation_threshold_action_enum
    END,
    halt_on_pending_reconciliation = require_reconciliation_to_close,
    enable_reconciliation_notifications = notify_on_reconciliation_exception,
    revenue_recognition_method = CASE
        WHEN revenue_recognition_policy = 'OnCashReceipt' THEN 'OnPayment'::revenue_recognition_enum
        ELSE 'OnBilling'::revenue_recognition_enum
    END,
    expense_recognition_method = CASE
        WHEN expense_recognition_policy = 'OnCashDisbursement' THEN 'OnPayment'::expense_recognition_enum
        ELSE 'OnAccrual'::expense_recognition_enum
    END,
    enable_multi_currency = (currency_mode = 'MultiCurrency'),
    default_currency_code = functional_currency_code,
    currency_gain_account_id = realized_fx_gain_account_id,
    currency_loss_account_id = realized_fx_loss_account_id;

ALTER TABLE accounting_controls
    DROP COLUMN IF EXISTS accounting_basis,
    DROP COLUMN IF EXISTS revenue_recognition_policy,
    DROP COLUMN IF EXISTS expense_recognition_policy,
    DROP COLUMN IF EXISTS journal_posting_mode,
    DROP COLUMN IF EXISTS auto_post_source_events,
    DROP COLUMN IF EXISTS manual_journal_entry_policy,
    DROP COLUMN IF EXISTS require_manual_je_approval,
    DROP COLUMN IF EXISTS journal_reversal_policy,
    DROP COLUMN IF EXISTS period_close_mode,
    DROP COLUMN IF EXISTS require_period_close_approval,
    DROP COLUMN IF EXISTS locked_period_posting_policy,
    DROP COLUMN IF EXISTS closed_period_posting_policy,
    DROP COLUMN IF EXISTS require_reconciliation_to_close,
    DROP COLUMN IF EXISTS reconciliation_mode,
    DROP COLUMN IF EXISTS reconciliation_tolerance_amount,
    DROP COLUMN IF EXISTS notify_on_reconciliation_exception,
    DROP COLUMN IF EXISTS currency_mode,
    DROP COLUMN IF EXISTS functional_currency_code,
    DROP COLUMN IF EXISTS exchange_rate_date_policy,
    DROP COLUMN IF EXISTS exchange_rate_override_policy,
    DROP COLUMN IF EXISTS default_tax_liability_account_id,
    DROP COLUMN IF EXISTS default_write_off_account_id,
    DROP COLUMN IF EXISTS realized_fx_gain_account_id,
    DROP COLUMN IF EXISTS realized_fx_loss_account_id;

--bun:split
ALTER TABLE billing_controls
    ADD COLUMN IF NOT EXISTS payment_term payment_term_enum NOT NULL DEFAULT 'Net30',
    ADD COLUMN IF NOT EXISTS show_invoice_due_date BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS invoice_terms TEXT,
    ADD COLUMN IF NOT EXISTS invoice_footer TEXT,
    ADD COLUMN IF NOT EXISTS show_amount_due BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS auto_transfer BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS transfer_schedule transfer_schedule_enum NOT NULL DEFAULT 'Continuous',
    ADD COLUMN IF NOT EXISTS transfer_batch_size INTEGER NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS auto_mark_ready_to_bill BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS enforce_customer_billing_req BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS validate_customer_rates BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS auto_bill BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS send_auto_bill_notifications BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS auto_bill_batch_size INTEGER NOT NULL DEFAULT 100,
    ADD COLUMN IF NOT EXISTS billing_exception_handling billing_exception_handling_enum NOT NULL DEFAULT 'Queue',
    ADD COLUMN IF NOT EXISTS rate_discrepancy_threshold NUMERIC(10, 2) NOT NULL DEFAULT 5.00,
    ADD COLUMN IF NOT EXISTS auto_resolve_minor_discrepancies BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS allow_invoice_consolidation BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS consolidation_period_days INTEGER NOT NULL DEFAULT 7,
    ADD COLUMN IF NOT EXISTS group_consolidated_invoices BOOLEAN NOT NULL DEFAULT FALSE;

--bun:split
UPDATE billing_controls
SET
    payment_term = default_payment_term,
    show_invoice_due_date = show_due_date_on_invoice,
    invoice_terms = default_invoice_terms,
    invoice_footer = default_invoice_footer,
    show_amount_due = show_balance_due_on_invoice,
    auto_transfer = (billing_queue_transfer_mode = 'AutomaticWhenReady'),
    transfer_schedule = COALESCE(billing_queue_transfer_schedule, 'Continuous'::transfer_schedule_enum),
    transfer_batch_size = COALESCE(billing_queue_transfer_batch_size, 100),
    auto_mark_ready_to_bill = (ready_to_bill_assignment_mode = 'AutomaticWhenEligible'),
    enforce_customer_billing_req = (shipment_billing_requirement_enforcement = 'Block'),
    validate_customer_rates = (rate_validation_enforcement <> 'Ignore'),
    auto_bill = (invoice_posting_mode = 'AutomaticWhenNoBlockingExceptions'),
    send_auto_bill_notifications = notify_on_auto_invoice_creation,
    auto_bill_batch_size = COALESCE(auto_invoice_batch_size, 100),
    billing_exception_handling = CASE
        WHEN billing_exception_disposition = 'ReturnToOperations' THEN 'Reject'::billing_exception_handling_enum
        ELSE 'Queue'::billing_exception_handling_enum
    END,
    rate_discrepancy_threshold = 5.00,
    auto_resolve_minor_discrepancies = FALSE;

ALTER TABLE billing_controls
    DROP COLUMN IF EXISTS default_payment_term,
    DROP COLUMN IF EXISTS default_invoice_terms,
    DROP COLUMN IF EXISTS default_invoice_footer,
    DROP COLUMN IF EXISTS show_due_date_on_invoice,
    DROP COLUMN IF EXISTS show_balance_due_on_invoice,
    DROP COLUMN IF EXISTS ready_to_bill_assignment_mode,
    DROP COLUMN IF EXISTS billing_queue_transfer_mode,
    DROP COLUMN IF EXISTS billing_queue_transfer_schedule,
    DROP COLUMN IF EXISTS billing_queue_transfer_batch_size,
    DROP COLUMN IF EXISTS invoice_draft_creation_mode,
    DROP COLUMN IF EXISTS invoice_posting_mode,
    DROP COLUMN IF EXISTS auto_invoice_batch_size,
    DROP COLUMN IF EXISTS notify_on_auto_invoice_creation,
    DROP COLUMN IF EXISTS shipment_billing_requirement_enforcement,
    DROP COLUMN IF EXISTS rate_validation_enforcement,
    DROP COLUMN IF EXISTS billing_exception_disposition,
    DROP COLUMN IF EXISTS notify_on_billing_exceptions,
    DROP COLUMN IF EXISTS rate_variance_tolerance_percent,
    DROP COLUMN IF EXISTS rate_variance_auto_resolution_mode;

--bun:split
DROP TRIGGER IF EXISTS invoice_adjustment_controls_update_timestamp_trigger ON invoice_adjustment_controls;
DROP FUNCTION IF EXISTS invoice_adjustment_controls_update_timestamp();
DROP TABLE IF EXISTS invoice_adjustment_controls;

DROP TYPE IF EXISTS adjustment_eligibility_policy_enum;
DROP TYPE IF EXISTS adjustment_accounting_date_policy_enum;
DROP TYPE IF EXISTS closed_period_adjustment_policy_enum;
DROP TYPE IF EXISTS requirement_policy_enum;
DROP TYPE IF EXISTS adjustment_attachment_policy_enum;
DROP TYPE IF EXISTS approval_policy_enum;
DROP TYPE IF EXISTS write_off_approval_policy_enum;
DROP TYPE IF EXISTS replacement_invoice_review_policy_enum;
DROP TYPE IF EXISTS customer_credit_balance_policy_enum;
DROP TYPE IF EXISTS over_credit_policy_enum;
DROP TYPE IF EXISTS superseded_invoice_visibility_policy_enum;
DROP TYPE IF EXISTS accounting_basis_enum;
DROP TYPE IF EXISTS revenue_recognition_policy_enum;
DROP TYPE IF EXISTS expense_recognition_policy_enum;
DROP TYPE IF EXISTS journal_posting_mode_enum;
DROP TYPE IF EXISTS journal_source_event_enum;
DROP TYPE IF EXISTS manual_journal_entry_policy_enum;
DROP TYPE IF EXISTS journal_reversal_policy_enum;
DROP TYPE IF EXISTS period_close_mode_enum;
DROP TYPE IF EXISTS locked_period_posting_policy_enum;
DROP TYPE IF EXISTS closed_period_posting_policy_enum;
DROP TYPE IF EXISTS reconciliation_mode_enum;
DROP TYPE IF EXISTS currency_mode_enum;
DROP TYPE IF EXISTS exchange_rate_date_policy_enum;
DROP TYPE IF EXISTS exchange_rate_override_policy_enum;
DROP TYPE IF EXISTS enforcement_level_enum;
DROP TYPE IF EXISTS billing_exception_disposition_enum;
DROP TYPE IF EXISTS ready_to_bill_assignment_mode_enum;
DROP TYPE IF EXISTS billing_queue_transfer_mode_enum;
DROP TYPE IF EXISTS invoice_draft_creation_mode_enum;
DROP TYPE IF EXISTS invoice_posting_mode_enum;
DROP TYPE IF EXISTS rate_variance_auto_resolution_mode_enum;
