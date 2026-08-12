-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260410171000_add_manual_journal_requests.tx.up.sql

CREATE TABLE IF NOT EXISTS manual_journal_requests(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "request_number" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "description" TEXT NOT NULL,
    "reason" TEXT,
    "accounting_date" INTEGER NOT NULL,
    "requested_fiscal_year_id" TEXT NOT NULL,
    "requested_fiscal_period_id" TEXT NOT NULL,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "total_debit_minor" INTEGER NOT NULL DEFAULT 0,
    "total_credit_minor" INTEGER NOT NULL DEFAULT 0,
    "approved_at" INTEGER,
    "approved_by_id" TEXT,
    "rejected_at" INTEGER,
    "rejected_by_id" TEXT,
    "rejection_reason" TEXT,
    "cancelled_at" INTEGER,
    "cancelled_by_id" TEXT,
    "cancel_reason" TEXT,
    "posted_batch_id" TEXT,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_manual_journal_requests PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_manual_journal_requests_number UNIQUE (organization_id, business_unit_id, request_number),
    CONSTRAINT fk_manual_journal_requests_fiscal_year FOREIGN KEY (requested_fiscal_year_id, organization_id, business_unit_id)
        REFERENCES fiscal_years(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_manual_journal_requests_fiscal_period FOREIGN KEY (requested_fiscal_period_id, organization_id, business_unit_id)
        REFERENCES fiscal_periods(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_manual_journal_requests_approved_by FOREIGN KEY (approved_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_manual_journal_requests_rejected_by FOREIGN KEY (rejected_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_manual_journal_requests_cancelled_by FOREIGN KEY (cancelled_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_manual_journal_requests_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_manual_journal_requests_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_manual_journal_request_totals_nonnegative CHECK (total_debit_minor >= 0 AND total_credit_minor >= 0)
);

--bun:split

CREATE TABLE IF NOT EXISTS manual_journal_request_lines(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "manual_journal_request_id" TEXT NOT NULL,
    "line_number" INTEGER NOT NULL,
    "gl_account_id" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "debit_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "credit_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "customer_id" TEXT,
    "location_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_manual_journal_request_lines PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_manual_journal_request_lines_number UNIQUE (manual_journal_request_id, organization_id, business_unit_id, line_number),
    CONSTRAINT fk_manual_journal_request_lines_request FOREIGN KEY (manual_journal_request_id, organization_id, business_unit_id)
        REFERENCES manual_journal_requests(id, organization_id, business_unit_id) ON DELETE CASCADE,
    CONSTRAINT fk_manual_journal_request_lines_gl_account FOREIGN KEY (gl_account_id, organization_id, business_unit_id)
        REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    CONSTRAINT fk_manual_journal_request_lines_customer FOREIGN KEY (customer_id, organization_id, business_unit_id)
        REFERENCES customers(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_manual_journal_request_lines_location FOREIGN KEY (location_id, organization_id, business_unit_id)
        REFERENCES locations(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT chk_manual_journal_request_lines_debit_or_credit CHECK ((debit_amount_minor > 0 AND credit_amount_minor = 0) OR
        (credit_amount_minor > 0 AND debit_amount_minor = 0))
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_manual_journal_requests_status
    ON manual_journal_requests (organization_id, business_unit_id, status, accounting_date);

--bun:split

CREATE INDEX IF NOT EXISTS idx_manual_journal_requests_fiscal_period
    ON manual_journal_requests (organization_id, business_unit_id, requested_fiscal_period_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_manual_journal_request_lines_request
    ON manual_journal_request_lines (organization_id, business_unit_id, manual_journal_request_id);
