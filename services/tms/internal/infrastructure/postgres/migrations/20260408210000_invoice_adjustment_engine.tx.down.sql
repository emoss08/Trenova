DROP INDEX IF EXISTS idx_invoice_adjustment_batch_items_batch;

--bun:split
DROP INDEX IF EXISTS idx_invoice_reconciliation_exceptions_adjustment;

--bun:split
DROP INDEX IF EXISTS idx_invoice_adjustment_lines_original_line;

--bun:split
DROP INDEX IF EXISTS idx_invoice_adjustments_status;

--bun:split
DROP INDEX IF EXISTS idx_invoice_adjustments_original_invoice;

--bun:split
ALTER TABLE invoices
    DROP CONSTRAINT IF EXISTS fk_invoices_correction_group;

--bun:split
DROP TABLE IF EXISTS invoice_adjustment_batch_items;

--bun:split
DROP TABLE IF EXISTS invoice_adjustment_batches;

--bun:split
DROP TABLE IF EXISTS invoice_reconciliation_exceptions;

--bun:split
DROP TABLE IF EXISTS invoice_adjustment_snapshots;

--bun:split
DROP TABLE IF EXISTS invoice_adjustment_lines;

--bun:split
DROP TABLE IF EXISTS invoice_adjustments;

--bun:split
DROP TABLE IF EXISTS invoice_correction_groups;

--bun:split
ALTER TABLE billing_queue_items
    DROP COLUMN IF EXISTS is_adjustment_origin,
    DROP COLUMN IF EXISTS source_invoice_id,
    DROP COLUMN IF EXISTS source_invoice_adjustment_id,
    DROP COLUMN IF EXISTS source_credit_memo_invoice_id,
    DROP COLUMN IF EXISTS correction_group_id,
    DROP COLUMN IF EXISTS rebill_strategy,
    DROP COLUMN IF EXISTS requires_replacement_review,
    DROP COLUMN IF EXISTS rerate_variance_percent,
    DROP COLUMN IF EXISTS adjustment_context;

--bun:split
ALTER TABLE invoices
    DROP COLUMN IF EXISTS applied_amount,
    DROP COLUMN IF EXISTS settlement_status,
    DROP COLUMN IF EXISTS dispute_status,
    DROP COLUMN IF EXISTS correction_group_id,
    DROP COLUMN IF EXISTS supersedes_invoice_id,
    DROP COLUMN IF EXISTS superseded_by_invoice_id,
    DROP COLUMN IF EXISTS source_invoice_adjustment_id,
    DROP COLUMN IF EXISTS is_adjustment_artifact;
