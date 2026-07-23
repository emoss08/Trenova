CREATE TABLE IF NOT EXISTS recurring_earnings(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Active',
    frequency VARCHAR(50) NOT NULL DEFAULT 'EverySettlement',
    description VARCHAR(255) NOT NULL,
    amount_minor BIGINT NOT NULL,
    total_cap_minor BIGINT,
    paid_to_date_minor BIGINT NOT NULL DEFAULT 0,
    start_date BIGINT NOT NULL,
    end_date BIGINT,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    created_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_recurring_earnings PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_recurring_earnings_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_recurring_earnings_cap CHECK (total_cap_minor IS NULL OR total_cap_minor > 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_recurring_earnings_worker ON recurring_earnings(organization_id, business_unit_id, worker_id, status);

--bun:split
ALTER TABLE driver_settlement_lines
    ADD COLUMN IF NOT EXISTS recurring_earning_id VARCHAR(100);

--bun:split
ALTER TABLE accounting_controls
    ADD COLUMN IF NOT EXISTS default_driver_reimbursement_account_id VARCHAR(100);
