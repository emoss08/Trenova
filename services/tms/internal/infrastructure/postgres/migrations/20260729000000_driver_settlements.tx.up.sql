ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'DriverSettlementPosted';

--bun:split
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'DriverSettlementVoided';

--bun:split
ALTER TYPE journal_source_event_enum ADD VALUE IF NOT EXISTS 'EscrowInterestAccrued';

--bun:split
ALTER TABLE accounting_controls
    ADD COLUMN IF NOT EXISTS default_driver_pay_expense_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_purchased_transportation_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_settlements_payable_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_driver_advance_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_escrow_liability_account_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS default_escrow_interest_expense_account_id VARCHAR(100);

--bun:split
ALTER TABLE tractors
    ADD COLUMN IF NOT EXISTS ownership_type VARCHAR(50) NOT NULL DEFAULT 'CompanyOwned',
    ADD COLUMN IF NOT EXISTS owner_worker_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS lessor_name VARCHAR(150),
    ADD COLUMN IF NOT EXISTS lease_reference VARCHAR(100),
    ADD COLUMN IF NOT EXISTS lease_end_date BIGINT;

--bun:split
ALTER TABLE tractors
    ADD CONSTRAINT fk_tractors_owner_worker FOREIGN KEY (owner_worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
ALTER TABLE trailers
    ADD COLUMN IF NOT EXISTS ownership_type VARCHAR(50) NOT NULL DEFAULT 'CompanyOwned',
    ADD COLUMN IF NOT EXISTS owner_worker_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS lessor_name VARCHAR(150),
    ADD COLUMN IF NOT EXISTS lease_reference VARCHAR(100),
    ADD COLUMN IF NOT EXISTS lease_end_date BIGINT;

--bun:split
ALTER TABLE trailers
    ADD CONSTRAINT fk_trailers_owner_worker FOREIGN KEY (owner_worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE RESTRICT;

--bun:split
CREATE TABLE IF NOT EXISTS settlement_controls(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    pay_period_frequency VARCHAR(50) NOT NULL DEFAULT 'Weekly',
    period_end_day_of_week INTEGER NOT NULL DEFAULT 6,
    pay_delay_days INTEGER NOT NULL DEFAULT 5,
    pay_trigger VARCHAR(50) NOT NULL DEFAULT 'ShipmentDelivered',
    auto_generate_batches BOOLEAN NOT NULL DEFAULT FALSE,
    auto_approve_clean BOOLEAN NOT NULL DEFAULT FALSE,
    allow_negative_net BOOLEAN NOT NULL DEFAULT TRUE,
    variance_threshold_pct NUMERIC(7, 4) NOT NULL DEFAULT 25,
    variance_lookback_weeks INTEGER NOT NULL DEFAULT 8,
    default_escrow_interest_rate NUMERIC(7, 4) NOT NULL DEFAULT 0,
    escrow_interest_frequency_months INTEGER NOT NULL DEFAULT 3,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_settlement_controls PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_settlement_controls_period_end_day CHECK (period_end_day_of_week BETWEEN 0 AND 6),
    CONSTRAINT chk_settlement_controls_pay_delay CHECK (pay_delay_days BETWEEN 0 AND 30),
    CONSTRAINT chk_settlement_controls_escrow_freq CHECK (escrow_interest_frequency_months BETWEEN 1 AND 3),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_settlement_controls_org ON settlement_controls(organization_id, business_unit_id);

--bun:split
CREATE TABLE IF NOT EXISTS driver_pay_profiles(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    status status_enum NOT NULL DEFAULT 'Active',
    name VARCHAR(100) NOT NULL,
    description TEXT,
    classification VARCHAR(50) NOT NULL DEFAULT 'CompanyDriver',
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    guaranteed_period_minimum_minor BIGINT NOT NULL DEFAULT 0,
    per_diem_rate_per_mile NUMERIC(19, 4) NOT NULL DEFAULT 0,
    per_diem_daily_cap_minor BIGINT NOT NULL DEFAULT 0,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_driver_pay_profiles PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_pay_profiles_guarantee CHECK (guaranteed_period_minimum_minor >= 0),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_driver_pay_profiles_name ON driver_pay_profiles(organization_id, business_unit_id, lower(name));

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_pay_profiles_status ON driver_pay_profiles(organization_id, business_unit_id, status);

--bun:split
CREATE TABLE IF NOT EXISTS driver_pay_profile_components(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    pay_profile_id VARCHAR(100) NOT NULL,
    kind VARCHAR(50) NOT NULL,
    method VARCHAR(50) NOT NULL,
    description VARCHAR(255),
    rate NUMERIC(19, 4) NOT NULL DEFAULT 0,
    revenue_basis VARCHAR(50),
    bands JSONB,
    free_time_minutes INTEGER NOT NULL DEFAULT 0,
    min_amount_minor BIGINT,
    max_amount_minor BIGINT,
    sequence INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_driver_pay_profile_components PRIMARY KEY (id, organization_id, business_unit_id),
    FOREIGN KEY (pay_profile_id, organization_id, business_unit_id) REFERENCES driver_pay_profiles(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_pay_profile_components_profile ON driver_pay_profile_components(organization_id, business_unit_id, pay_profile_id);

--bun:split
CREATE TABLE IF NOT EXISTS worker_pay_assignments(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    pay_profile_id VARCHAR(100) NOT NULL,
    effective_from BIGINT NOT NULL,
    effective_to BIGINT,
    split_percent NUMERIC(7, 4) NOT NULL DEFAULT 100,
    notes TEXT,
    created_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_worker_pay_assignments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_worker_pay_assignments_split CHECK (split_percent > 0 AND split_percent <= 100),
    CONSTRAINT chk_worker_pay_assignments_range CHECK (effective_to IS NULL OR effective_to > effective_from),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (pay_profile_id, organization_id, business_unit_id) REFERENCES driver_pay_profiles(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_worker_pay_assignments_worker ON worker_pay_assignments(organization_id, business_unit_id, worker_id, effective_from DESC);

--bun:split
CREATE TABLE IF NOT EXISTS escrow_accounts(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Active',
    target_amount_minor BIGINT NOT NULL DEFAULT 0,
    balance_minor BIGINT NOT NULL DEFAULT 0,
    annual_interest_rate NUMERIC(7, 4) NOT NULL DEFAULT 0,
    last_interest_accrual_date BIGINT,
    opened_date BIGINT NOT NULL,
    closed_date BIGINT,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_escrow_accounts PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_escrow_accounts_target CHECK (target_amount_minor >= 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE RESTRICT
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_escrow_accounts_worker_active ON escrow_accounts(organization_id, business_unit_id, worker_id) WHERE status = 'Active';

--bun:split
CREATE TABLE IF NOT EXISTS escrow_transactions(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    escrow_account_id VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    amount_minor BIGINT NOT NULL,
    balance_after_minor BIGINT NOT NULL,
    occurred_date BIGINT NOT NULL,
    description VARCHAR(255),
    settlement_id VARCHAR(100),
    created_by_id VARCHAR(100),
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_escrow_transactions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_escrow_transactions_amount CHECK (amount_minor <> 0),
    FOREIGN KEY (escrow_account_id, organization_id, business_unit_id) REFERENCES escrow_accounts(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_escrow_transactions_account ON escrow_transactions(organization_id, business_unit_id, escrow_account_id, occurred_date DESC);

--bun:split
CREATE TABLE IF NOT EXISTS recurring_deductions(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    escrow_account_id VARCHAR(100),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Active',
    frequency VARCHAR(50) NOT NULL DEFAULT 'EverySettlement',
    description VARCHAR(255) NOT NULL,
    amount_minor BIGINT NOT NULL,
    total_cap_minor BIGINT,
    deducted_to_date_minor BIGINT NOT NULL DEFAULT 0,
    start_date BIGINT NOT NULL,
    end_date BIGINT,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    created_by_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_recurring_deductions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_recurring_deductions_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_recurring_deductions_cap CHECK (total_cap_minor IS NULL OR total_cap_minor > 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (escrow_account_id, organization_id, business_unit_id) REFERENCES escrow_accounts(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_recurring_deductions_worker ON recurring_deductions(organization_id, business_unit_id, worker_id, status);

--bun:split
CREATE TABLE IF NOT EXISTS pay_advances(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Outstanding',
    source VARCHAR(50) NOT NULL,
    reference VARCHAR(100),
    issued_date BIGINT NOT NULL,
    amount_minor BIGINT NOT NULL,
    recovered_minor BIGINT NOT NULL DEFAULT 0,
    written_off_minor BIGINT NOT NULL DEFAULT 0,
    write_off_reason TEXT,
    notes TEXT,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    created_by_id VARCHAR(100),
    written_off_by_id VARCHAR(100),
    written_off_at BIGINT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_pay_advances PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_pay_advances_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_pay_advances_recovery CHECK (recovered_minor >= 0 AND written_off_minor >= 0 AND recovered_minor + written_off_minor <= amount_minor),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (written_off_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_pay_advances_worker ON pay_advances(organization_id, business_unit_id, worker_id, status);

--bun:split
CREATE TABLE IF NOT EXISTS driver_settlement_batches(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Open',
    name VARCHAR(100) NOT NULL,
    period_start BIGINT NOT NULL,
    period_end BIGINT NOT NULL,
    pay_date BIGINT NOT NULL,
    settlement_count INTEGER NOT NULL DEFAULT 0,
    exception_count INTEGER NOT NULL DEFAULT 0,
    total_gross_minor BIGINT NOT NULL DEFAULT 0,
    total_net_minor BIGINT NOT NULL DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    notes TEXT,
    generated_by_id VARCHAR(100),
    generated_at BIGINT,
    completed_at BIGINT,
    canceled_by_id VARCHAR(100),
    canceled_at BIGINT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_driver_settlement_batches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_settlement_batches_period CHECK (period_end > period_start),
    FOREIGN KEY (generated_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (canceled_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_settlement_batches_status ON driver_settlement_batches(organization_id, business_unit_id, status, period_end DESC);

--bun:split
CREATE TABLE IF NOT EXISTS driver_settlements(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    batch_id VARCHAR(100),
    pay_profile_id VARCHAR(100),
    settlement_number VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Draft',
    classification VARCHAR(50) NOT NULL,
    pay_profile_name VARCHAR(100),
    period_start BIGINT NOT NULL,
    period_end BIGINT NOT NULL,
    pay_date BIGINT NOT NULL,
    gross_earnings_minor BIGINT NOT NULL DEFAULT 0,
    reimbursements_minor BIGINT NOT NULL DEFAULT 0,
    deductions_minor BIGINT NOT NULL DEFAULT 0,
    carry_forward_in_minor BIGINT NOT NULL DEFAULT 0,
    carry_forward_out_minor BIGINT NOT NULL DEFAULT 0,
    net_pay_minor BIGINT NOT NULL DEFAULT 0,
    total_miles NUMERIC(19, 4) NOT NULL DEFAULT 0,
    shipment_count INTEGER NOT NULL DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    has_exceptions BOOLEAN NOT NULL DEFAULT FALSE,
    exceptions JSONB,
    notes TEXT,
    submitted_by_id VARCHAR(100),
    submitted_at BIGINT,
    approved_by_id VARCHAR(100),
    approved_at BIGINT,
    posted_by_id VARCHAR(100),
    posted_at BIGINT,
    posted_journal_batch_id VARCHAR(100),
    paid_at BIGINT,
    paid_by_id VARCHAR(100),
    payment_method VARCHAR(50),
    payment_reference VARCHAR(100),
    voided_by_id VARCHAR(100),
    voided_at BIGINT,
    void_reason TEXT,
    void_journal_batch_id VARCHAR(100),
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_driver_settlements PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_settlements_period CHECK (period_end > period_start),
    CONSTRAINT chk_driver_settlements_net CHECK (net_pay_minor >= 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (batch_id, organization_id, business_unit_id) REFERENCES driver_settlement_batches(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (pay_profile_id, organization_id, business_unit_id) REFERENCES driver_pay_profiles(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (submitted_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (approved_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (posted_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (paid_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (voided_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_driver_settlements_number ON driver_settlements(organization_id, business_unit_id, settlement_number);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_settlements_worker ON driver_settlements(organization_id, business_unit_id, worker_id, period_end DESC);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_settlements_status ON driver_settlements(organization_id, business_unit_id, status);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_settlements_batch ON driver_settlements(organization_id, business_unit_id, batch_id) WHERE batch_id IS NOT NULL;

--bun:split
CREATE TABLE IF NOT EXISTS driver_settlement_lines(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    settlement_id VARCHAR(100) NOT NULL,
    line_number INTEGER NOT NULL,
    category VARCHAR(50) NOT NULL,
    component_kind VARCHAR(50),
    method VARCHAR(50),
    description VARCHAR(255) NOT NULL,
    quantity NUMERIC(19, 4) NOT NULL DEFAULT 0,
    rate NUMERIC(19, 4) NOT NULL DEFAULT 0,
    amount_minor BIGINT NOT NULL,
    shipment_id VARCHAR(100),
    move_id VARCHAR(100),
    pay_event_id VARCHAR(100),
    recurring_deduction_id VARCHAR(100),
    advance_id VARCHAR(100),
    escrow_account_id VARCHAR(100),
    pro_number VARCHAR(100),
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_driver_settlement_lines PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_driver_settlement_lines_number UNIQUE (settlement_id, organization_id, business_unit_id, line_number),
    FOREIGN KEY (settlement_id, organization_id, business_unit_id) REFERENCES driver_settlements(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_settlement_lines_settlement ON driver_settlement_lines(organization_id, business_unit_id, settlement_id);

--bun:split
CREATE TABLE IF NOT EXISTS driver_pay_events(
    id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    worker_id VARCHAR(100) NOT NULL,
    shipment_id VARCHAR(100) NOT NULL,
    move_id VARCHAR(100),
    assignment_id VARCHAR(100),
    pay_profile_id VARCHAR(100),
    settlement_id VARCHAR(100),
    settlement_line_id VARCHAR(100),
    idempotency_key VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'Accrued',
    event_date BIGINT NOT NULL,
    gross_amount_minor BIGINT NOT NULL DEFAULT 0,
    total_miles NUMERIC(19, 4) NOT NULL DEFAULT 0,
    currency_code VARCHAR(3) NOT NULL DEFAULT 'USD',
    components JSONB,
    pro_number VARCHAR(100),
    voided_at BIGINT,
    void_reason TEXT,
    version BIGINT NOT NULL DEFAULT 0,
    created_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    updated_at BIGINT NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT pk_driver_pay_events PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_pay_events_gross CHECK (gross_amount_minor >= 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (shipment_id, organization_id, business_unit_id) REFERENCES shipments(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS uq_driver_pay_events_idempotency ON driver_pay_events(organization_id, business_unit_id, idempotency_key);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_pay_events_worker ON driver_pay_events(organization_id, business_unit_id, worker_id, status, event_date DESC);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_pay_events_shipment ON driver_pay_events(organization_id, business_unit_id, shipment_id);

--bun:split
CREATE INDEX IF NOT EXISTS idx_driver_pay_events_settlement ON driver_pay_events(organization_id, business_unit_id, settlement_id) WHERE settlement_id IS NOT NULL;
