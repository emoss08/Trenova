-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260408210000_invoice_adjustment_engine.tx.up.sql

ALTER TABLE "invoices" ADD COLUMN "applied_amount" NUMERIC NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "settlement_status" TEXT NOT NULL DEFAULT 'Unpaid';

--bun:split

ALTER TABLE "invoices" ADD COLUMN "dispute_status" TEXT NOT NULL DEFAULT 'None';

--bun:split

ALTER TABLE "invoices" ADD COLUMN "correction_group_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "supersedes_invoice_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "superseded_by_invoice_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "source_invoice_adjustment_id" TEXT;

--bun:split

ALTER TABLE "invoices" ADD COLUMN "is_adjustment_artifact" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "is_adjustment_origin" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "source_invoice_id" TEXT;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "source_invoice_adjustment_id" TEXT;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "source_credit_memo_invoice_id" TEXT;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "correction_group_id" TEXT;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "rebill_strategy" TEXT;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "requires_replacement_review" INTEGER NOT NULL DEFAULT 0;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "rerate_variance_percent" NUMERIC;

--bun:split

ALTER TABLE "billing_queue_items" ADD COLUMN "adjustment_context" TEXT NOT NULL DEFAULT '{}';

--bun:split

CREATE TABLE IF NOT EXISTS invoice_correction_groups(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "root_invoice_id" TEXT NOT NULL,
    "current_invoice_id" TEXT,
    "metadata" TEXT NOT NULL DEFAULT '{}',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_correction_groups PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_correction_groups_root_invoice FOREIGN KEY (root_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_invoice_correction_groups_current_invoice FOREIGN KEY (current_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_adjustments(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "correction_group_id" TEXT NOT NULL,
    "original_invoice_id" TEXT NOT NULL,
    "credit_memo_invoice_id" TEXT,
    "replacement_invoice_id" TEXT,
    "rebill_queue_item_id" TEXT,
    "batch_id" TEXT,
    "kind" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "approval_status" TEXT NOT NULL DEFAULT 'NotRequired',
    "replacement_review_status" TEXT NOT NULL DEFAULT 'NotRequired',
    "rebill_strategy" TEXT,
    "reason" TEXT,
    "policy_reason" TEXT,
    "idempotency_key" TEXT NOT NULL,
    "accounting_date" INTEGER NOT NULL,
    "credit_total_amount" NUMERIC NOT NULL DEFAULT 0,
    "rebill_total_amount" NUMERIC NOT NULL DEFAULT 0,
    "net_delta_amount" NUMERIC NOT NULL DEFAULT 0,
    "rerate_variance_percent" NUMERIC NOT NULL DEFAULT 0,
    "would_create_unapplied_credit" INTEGER NOT NULL DEFAULT 0,
    "requires_reconciliation_exception" INTEGER NOT NULL DEFAULT 0,
    "approval_required" INTEGER NOT NULL DEFAULT 0,
    "submitted_by_id" TEXT,
    "submitted_at" INTEGER,
    "approved_by_id" TEXT,
    "approved_at" INTEGER,
    "rejected_by_id" TEXT,
    "rejected_at" INTEGER,
    "rejection_reason" TEXT,
    "execution_error" TEXT,
    "metadata" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_adjustments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_invoice_adjustments_idempotency UNIQUE (organization_id, business_unit_id, idempotency_key),
    CONSTRAINT fk_invoice_adjustments_correction_group FOREIGN KEY (correction_group_id, organization_id, business_unit_id)
        REFERENCES invoice_correction_groups(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_invoice_adjustments_original_invoice FOREIGN KEY (original_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_invoice_adjustments_credit_memo_invoice FOREIGN KEY (credit_memo_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_invoice_adjustments_replacement_invoice FOREIGN KEY (replacement_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_invoice_adjustments_rebill_queue_item FOREIGN KEY (rebill_queue_item_id, organization_id, business_unit_id)
        REFERENCES billing_queue_items(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_adjustment_lines(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "adjustment_id" TEXT NOT NULL,
    "original_invoice_id" TEXT NOT NULL,
    "original_line_id" TEXT NOT NULL,
    "credit_memo_line_id" TEXT,
    "replacement_line_id" TEXT,
    "line_number" INTEGER NOT NULL,
    "description" TEXT NOT NULL,
    "credit_quantity" NUMERIC NOT NULL DEFAULT 0,
    "credit_amount" NUMERIC NOT NULL DEFAULT 0,
    "remaining_eligible_amount" NUMERIC NOT NULL DEFAULT 0,
    "rebill_quantity" NUMERIC NOT NULL DEFAULT 0,
    "rebill_amount" NUMERIC NOT NULL DEFAULT 0,
    "replacement_payload" TEXT NOT NULL DEFAULT '{}',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_adjustment_lines PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_adjustment_lines_adjustment FOREIGN KEY (adjustment_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_adjustment_lines_original_invoice FOREIGN KEY (original_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_invoice_adjustment_lines_original_line FOREIGN KEY (original_line_id, organization_id, business_unit_id)
        REFERENCES invoice_lines(id, organization_id, business_unit_id) ON DELETE RESTRICT
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_adjustment_snapshots(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "adjustment_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "payload" TEXT NOT NULL DEFAULT '{}',
    "created_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_adjustment_snapshots PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_adjustment_snapshots_adjustment FOREIGN KEY (adjustment_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_adjustment_snapshots_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_reconciliation_exceptions(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "adjustment_id" TEXT NOT NULL,
    "invoice_id" TEXT NOT NULL,
    "credit_memo_invoice_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "reason" TEXT NOT NULL,
    "amount" NUMERIC NOT NULL DEFAULT 0,
    "metadata" TEXT NOT NULL DEFAULT '{}',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_reconciliation_exceptions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_reconciliation_exceptions_adjustment FOREIGN KEY (adjustment_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustments(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_adjustment_batches(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "idempotency_key" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "total_count" INTEGER NOT NULL DEFAULT 0,
    "processed_count" INTEGER NOT NULL DEFAULT 0,
    "succeeded_count" INTEGER NOT NULL DEFAULT 0,
    "failed_count" INTEGER NOT NULL DEFAULT 0,
    "submitted_by_id" TEXT,
    "submitted_at" INTEGER,
    "metadata" TEXT NOT NULL DEFAULT '{}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_adjustment_batches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_invoice_adjustment_batches_idempotency UNIQUE (organization_id, business_unit_id, idempotency_key)
);

--bun:split

CREATE TABLE IF NOT EXISTS invoice_adjustment_batch_items(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "batch_id" TEXT NOT NULL,
    "adjustment_id" TEXT,
    "invoice_id" TEXT NOT NULL,
    "idempotency_key" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "error_message" TEXT,
    "request_payload" TEXT NOT NULL DEFAULT '{}',
    "result_payload" TEXT NOT NULL DEFAULT '{}',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_invoice_adjustment_batch_items PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_invoice_adjustment_batch_items_idempotency UNIQUE (organization_id, business_unit_id, idempotency_key),
    CONSTRAINT fk_invoice_adjustment_batch_items_batch FOREIGN KEY (batch_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustment_batches(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustments_original_invoice
    ON invoice_adjustments (organization_id, business_unit_id, original_invoice_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustments_status
    ON invoice_adjustments (organization_id, business_unit_id, status, approval_status);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_lines_original_line
    ON invoice_adjustment_lines (organization_id, business_unit_id, original_line_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_reconciliation_exceptions_adjustment
    ON invoice_reconciliation_exceptions (organization_id, business_unit_id, adjustment_id, status);

--bun:split

CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_batch_items_batch
    ON invoice_adjustment_batch_items (organization_id, business_unit_id, batch_id, status);
