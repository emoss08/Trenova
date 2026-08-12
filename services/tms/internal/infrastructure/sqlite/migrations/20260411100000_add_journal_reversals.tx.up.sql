-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260411100000_add_journal_reversals.tx.up.sql

CREATE TABLE IF NOT EXISTS journal_reversals(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "original_journal_entry_id" TEXT NOT NULL,
    "reversal_journal_entry_id" TEXT,
    "posted_batch_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Requested',
    "requested_accounting_date" INTEGER NOT NULL,
    "resolved_fiscal_year_id" TEXT NOT NULL,
    "resolved_fiscal_period_id" TEXT NOT NULL,
    "reason_code" TEXT NOT NULL,
    "reason_text" TEXT NOT NULL,
    "requested_by_id" TEXT NOT NULL,
    "approved_by_id" TEXT,
    "approved_at" INTEGER,
    "rejected_by_id" TEXT,
    "rejected_at" INTEGER,
    "rejection_reason" TEXT,
    "cancelled_by_id" TEXT,
    "cancelled_at" INTEGER,
    "cancel_reason" TEXT,
    "posted_by_id" TEXT,
    "posted_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
    ON journal_reversals (organization_id, business_unit_id, original_journal_entry_id)WHERE status IN ('Requested', 'PendingApproval', 'Approved', 'Posted');
