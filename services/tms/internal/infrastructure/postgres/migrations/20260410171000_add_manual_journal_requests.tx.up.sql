CREATE TYPE manual_journal_request_status_enum AS ENUM (
    'Draft',
    'PendingApproval',
    'Approved',
    'Rejected',
    'Cancelled',
    'Posted'
);

--bun:split
CREATE TABLE IF NOT EXISTS manual_journal_requests(
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    request_number VARCHAR(50) NOT NULL,
    status manual_journal_request_status_enum NOT NULL DEFAULT 'Draft',
    description TEXT NOT NULL,
    reason TEXT,
    accounting_date BIGINT NOT NULL,
    requested_fiscal_year_id VARCHAR(100) NOT NULL,
    requested_fiscal_period_id VARCHAR(100) NOT NULL,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    total_debit_minor BIGINT NOT NULL DEFAULT 0,
    total_credit_minor BIGINT NOT NULL DEFAULT 0,
    approved_at BIGINT,
    approved_by_id VARCHAR(100),
    rejected_at BIGINT,
    rejected_by_id VARCHAR(100),
    rejection_reason TEXT,
    cancelled_at BIGINT,
    cancelled_by_id VARCHAR(100),
    cancel_reason TEXT,
    posted_batch_id VARCHAR(100),
    created_by_id VARCHAR(100) NOT NULL,
    updated_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
    id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    manual_journal_request_id VARCHAR(100) NOT NULL,
    line_number INTEGER NOT NULL,
    gl_account_id VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    debit_amount_minor BIGINT NOT NULL DEFAULT 0,
    credit_amount_minor BIGINT NOT NULL DEFAULT 0,
    customer_id VARCHAR(100),
    location_id VARCHAR(100),
    created_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
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
    CONSTRAINT chk_manual_journal_request_lines_debit_or_credit CHECK (
        (debit_amount_minor > 0 AND credit_amount_minor = 0) OR
        (credit_amount_minor > 0 AND debit_amount_minor = 0)
    )
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_manual_journal_requests_status
    ON manual_journal_requests(organization_id, business_unit_id, status, accounting_date);

--bun:split
CREATE INDEX IF NOT EXISTS idx_manual_journal_requests_fiscal_period
    ON manual_journal_requests(organization_id, business_unit_id, requested_fiscal_period_id);

--bun:split
CREATE INDEX IF NOT EXISTS idx_manual_journal_request_lines_request
    ON manual_journal_request_lines(organization_id, business_unit_id, manual_journal_request_id);
