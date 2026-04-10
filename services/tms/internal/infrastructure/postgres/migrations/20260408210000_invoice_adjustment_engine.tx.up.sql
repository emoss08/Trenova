ALTER TABLE invoices
    ADD COLUMN IF NOT EXISTS applied_amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS settlement_status VARCHAR(50) NOT NULL DEFAULT 'Unpaid',
    ADD COLUMN IF NOT EXISTS dispute_status VARCHAR(50) NOT NULL DEFAULT 'None',
    ADD COLUMN IF NOT EXISTS correction_group_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS supersedes_invoice_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS superseded_by_invoice_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS source_invoice_adjustment_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS is_adjustment_artifact BOOLEAN NOT NULL DEFAULT FALSE;

--bun:split
ALTER TABLE billing_queue_items
    ADD COLUMN IF NOT EXISTS is_adjustment_origin BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS source_invoice_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS source_invoice_adjustment_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS source_credit_memo_invoice_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS correction_group_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS rebill_strategy VARCHAR(50),
    ADD COLUMN IF NOT EXISTS requires_replacement_review BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS rerate_variance_percent NUMERIC(9,6),
    ADD COLUMN IF NOT EXISTS adjustment_context JSONB NOT NULL DEFAULT '{}'::jsonb;

--bun:split
CREATE TABLE IF NOT EXISTS invoice_correction_groups(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    root_invoice_id VARCHAR(100) NOT NULL,
    current_invoice_id VARCHAR(100),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_correction_groups PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_correction_groups_root_invoice FOREIGN KEY (root_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_invoice_correction_groups_current_invoice FOREIGN KEY (current_invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split
CREATE TABLE IF NOT EXISTS invoice_adjustments(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    correction_group_id VARCHAR(100) NOT NULL,
    original_invoice_id VARCHAR(100) NOT NULL,
    credit_memo_invoice_id VARCHAR(100),
    replacement_invoice_id VARCHAR(100),
    rebill_queue_item_id VARCHAR(100),
    batch_id VARCHAR(100),
    kind VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL,
    approval_status VARCHAR(50) NOT NULL DEFAULT 'NotRequired',
    replacement_review_status VARCHAR(50) NOT NULL DEFAULT 'NotRequired',
    rebill_strategy VARCHAR(50),
    reason TEXT,
    policy_reason TEXT,
    idempotency_key VARCHAR(200) NOT NULL,
    accounting_date BIGINT NOT NULL,
    credit_total_amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    rebill_total_amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    net_delta_amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    rerate_variance_percent NUMERIC(9,6) NOT NULL DEFAULT 0,
    would_create_unapplied_credit BOOLEAN NOT NULL DEFAULT FALSE,
    requires_reconciliation_exception BOOLEAN NOT NULL DEFAULT FALSE,
    approval_required BOOLEAN NOT NULL DEFAULT FALSE,
    submitted_by_id VARCHAR(100),
    submitted_at BIGINT,
    approved_by_id VARCHAR(100),
    approved_at BIGINT,
    rejected_by_id VARCHAR(100),
    rejected_at BIGINT,
    rejection_reason TEXT,
    execution_error TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    adjustment_id VARCHAR(100) NOT NULL,
    original_invoice_id VARCHAR(100) NOT NULL,
    original_line_id VARCHAR(100) NOT NULL,
    credit_memo_line_id VARCHAR(100),
    replacement_line_id VARCHAR(100),
    line_number INTEGER NOT NULL,
    description TEXT NOT NULL,
    credit_quantity NUMERIC(19,4) NOT NULL DEFAULT 0,
    credit_amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    remaining_eligible_amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    rebill_quantity NUMERIC(19,4) NOT NULL DEFAULT 0,
    rebill_amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    replacement_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    adjustment_id VARCHAR(100) NOT NULL,
    invoice_id VARCHAR(100) NOT NULL,
    kind VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by_id VARCHAR(100),
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_adjustment_snapshots PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_adjustment_snapshots_adjustment FOREIGN KEY (adjustment_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustments(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_invoice_adjustment_snapshots_invoice FOREIGN KEY (invoice_id, organization_id, business_unit_id)
        REFERENCES invoices(id, organization_id, business_unit_id) ON DELETE RESTRICT
);

--bun:split
CREATE TABLE IF NOT EXISTS invoice_reconciliation_exceptions(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    adjustment_id VARCHAR(100) NOT NULL,
    invoice_id VARCHAR(100) NOT NULL,
    credit_memo_invoice_id VARCHAR(100),
    status VARCHAR(50) NOT NULL DEFAULT 'Open',
    reason TEXT NOT NULL,
    amount NUMERIC(19,4) NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_reconciliation_exceptions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_invoice_reconciliation_exceptions_adjustment FOREIGN KEY (adjustment_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustments(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE TABLE IF NOT EXISTS invoice_adjustment_batches(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    idempotency_key VARCHAR(200) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending',
    total_count INTEGER NOT NULL DEFAULT 0,
    processed_count INTEGER NOT NULL DEFAULT 0,
    succeeded_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    submitted_by_id VARCHAR(100),
    submitted_at BIGINT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_adjustment_batches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_invoice_adjustment_batches_idempotency UNIQUE (organization_id, business_unit_id, idempotency_key)
);

--bun:split
CREATE TABLE IF NOT EXISTS invoice_adjustment_batch_items(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    batch_id VARCHAR(100) NOT NULL,
    adjustment_id VARCHAR(100),
    invoice_id VARCHAR(100) NOT NULL,
    idempotency_key VARCHAR(200) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Pending',
    error_message TEXT,
    request_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    result_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_invoice_adjustment_batch_items PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_invoice_adjustment_batch_items_idempotency UNIQUE (organization_id, business_unit_id, idempotency_key),
    CONSTRAINT fk_invoice_adjustment_batch_items_batch FOREIGN KEY (batch_id, organization_id, business_unit_id)
        REFERENCES invoice_adjustment_batches(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
ALTER TABLE invoices
    ADD CONSTRAINT fk_invoices_correction_group
        FOREIGN KEY (correction_group_id, organization_id, business_unit_id)
        REFERENCES invoice_correction_groups(id, organization_id, business_unit_id)
        ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoice_adjustments_original_invoice
    ON invoice_adjustments(organization_id, business_unit_id, original_invoice_id);

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoice_adjustments_status
    ON invoice_adjustments(organization_id, business_unit_id, status, approval_status);

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_lines_original_line
    ON invoice_adjustment_lines(organization_id, business_unit_id, original_line_id);

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoice_reconciliation_exceptions_adjustment
    ON invoice_reconciliation_exceptions(organization_id, business_unit_id, adjustment_id, status);

--bun:split
CREATE INDEX IF NOT EXISTS idx_invoice_adjustment_batch_items_batch
    ON invoice_adjustment_batch_items(organization_id, business_unit_id, batch_id, status);
