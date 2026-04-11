CREATE TYPE journal_reversal_status_enum AS ENUM (
    'Requested',
    'PendingApproval',
    'Approved',
    'Rejected',
    'Cancelled',
    'Posted'
);

--bun:split
CREATE TABLE IF NOT EXISTS journal_reversals(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    original_journal_entry_id VARCHAR(100) NOT NULL,
    reversal_journal_entry_id VARCHAR(100),
    posted_batch_id VARCHAR(100),
    status journal_reversal_status_enum NOT NULL DEFAULT 'Requested',
    requested_accounting_date BIGINT NOT NULL,
    resolved_fiscal_year_id VARCHAR(100) NOT NULL,
    resolved_fiscal_period_id VARCHAR(100) NOT NULL,
    reason_code VARCHAR(50) NOT NULL,
    reason_text TEXT NOT NULL,
    requested_by_id VARCHAR(100) NOT NULL,
    approved_by_id VARCHAR(100),
    approved_at BIGINT,
    rejected_by_id VARCHAR(100),
    rejected_at BIGINT,
    rejection_reason TEXT,
    cancelled_by_id VARCHAR(100),
    cancelled_at BIGINT,
    cancel_reason TEXT,
    posted_by_id VARCHAR(100),
    posted_at BIGINT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT pk_journal_reversals PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_journal_reversals_original_entry FOREIGN KEY (original_journal_entry_id, organization_id, business_unit_id) REFERENCES journal_entries(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_journal_reversals_reversal_entry FOREIGN KEY (reversal_journal_entry_id, organization_id, business_unit_id) REFERENCES journal_entries(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_journal_reversals_batch FOREIGN KEY (posted_batch_id, organization_id, business_unit_id) REFERENCES journal_batches(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_journal_reversals_fiscal_year FOREIGN KEY (resolved_fiscal_year_id, organization_id, business_unit_id) REFERENCES fiscal_years(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_journal_reversals_fiscal_period FOREIGN KEY (resolved_fiscal_period_id, organization_id, business_unit_id) REFERENCES fiscal_periods(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_journal_reversals_requested_by FOREIGN KEY (requested_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_journal_reversals_approved_by FOREIGN KEY (approved_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_journal_reversals_rejected_by FOREIGN KEY (rejected_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_journal_reversals_cancelled_by FOREIGN KEY (cancelled_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_journal_reversals_posted_by FOREIGN KEY (posted_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_journal_reversals_original_active
    ON journal_reversals(organization_id, business_unit_id, original_journal_entry_id)
    WHERE status IN ('Requested', 'PendingApproval', 'Approved', 'Posted');
