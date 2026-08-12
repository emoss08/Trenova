-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260729000000_driver_settlements.tx.up.sql

ALTER TABLE "accounting_controls" ADD COLUMN "default_driver_pay_expense_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_purchased_transportation_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_settlements_payable_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_driver_advance_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_escrow_liability_account_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_escrow_interest_expense_account_id" TEXT;

--bun:split

ALTER TABLE "tractors" ADD COLUMN "ownership_type" TEXT NOT NULL DEFAULT 'CompanyOwned';

--bun:split

ALTER TABLE "tractors" ADD COLUMN "owner_worker_id" TEXT;

--bun:split

ALTER TABLE "tractors" ADD COLUMN "lessor_name" TEXT;

--bun:split

ALTER TABLE "tractors" ADD COLUMN "lease_reference" TEXT;

--bun:split

ALTER TABLE "tractors" ADD COLUMN "lease_end_date" INTEGER;

--bun:split

ALTER TABLE "trailers" ADD COLUMN "ownership_type" TEXT NOT NULL DEFAULT 'CompanyOwned';

--bun:split

ALTER TABLE "trailers" ADD COLUMN "owner_worker_id" TEXT;

--bun:split

ALTER TABLE "trailers" ADD COLUMN "lessor_name" TEXT;

--bun:split

ALTER TABLE "trailers" ADD COLUMN "lease_reference" TEXT;

--bun:split

ALTER TABLE "trailers" ADD COLUMN "lease_end_date" INTEGER;

--bun:split

CREATE TABLE IF NOT EXISTS settlement_controls(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "pay_period_frequency" TEXT NOT NULL DEFAULT 'Weekly',
    "period_end_day_of_week" INTEGER NOT NULL DEFAULT 6,
    "pay_delay_days" INTEGER NOT NULL DEFAULT 5,
    "pay_trigger" TEXT NOT NULL DEFAULT 'ShipmentDelivered',
    "auto_generate_batches" INTEGER NOT NULL DEFAULT 0,
    "auto_approve_clean" INTEGER NOT NULL DEFAULT 0,
    "allow_negative_net" INTEGER NOT NULL DEFAULT 1,
    "variance_threshold_pct" REAL NOT NULL DEFAULT 25,
    "variance_lookback_weeks" INTEGER NOT NULL DEFAULT 8,
    "default_escrow_interest_rate" REAL NOT NULL DEFAULT 0,
    "escrow_interest_frequency_months" INTEGER NOT NULL DEFAULT 3,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_settlement_controls PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_settlement_controls_period_end_day CHECK (period_end_day_of_week BETWEEN 0 AND 6),
    CONSTRAINT chk_settlement_controls_pay_delay CHECK (pay_delay_days BETWEEN 0 AND 30),
    CONSTRAINT chk_settlement_controls_escrow_freq CHECK (escrow_interest_frequency_months BETWEEN 1 AND 3),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_settlement_controls_org ON settlement_controls (organization_id, business_unit_id);

--bun:split

CREATE TABLE IF NOT EXISTS driver_pay_profiles(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "name" TEXT NOT NULL,
    "description" TEXT,
    "classification" TEXT NOT NULL DEFAULT 'CompanyDriver',
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "guaranteed_period_minimum_minor" INTEGER NOT NULL DEFAULT 0,
    "per_diem_rate_per_mile" REAL NOT NULL DEFAULT 0,
    "per_diem_daily_cap_minor" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_driver_pay_profiles PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_pay_profiles_guarantee CHECK (guaranteed_period_minimum_minor >= 0),
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (business_unit_id) REFERENCES business_units(id) ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_driver_pay_profiles_name ON driver_pay_profiles (organization_id, business_unit_id, lower(name));

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_pay_profiles_status ON driver_pay_profiles (organization_id, business_unit_id, status);

--bun:split

CREATE TABLE IF NOT EXISTS driver_pay_profile_components(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "pay_profile_id" TEXT NOT NULL,
    "kind" TEXT NOT NULL,
    "method" TEXT NOT NULL,
    "description" TEXT,
    "rate" REAL NOT NULL DEFAULT 0,
    "revenue_basis" TEXT,
    "bands" TEXT,
    "free_time_minutes" INTEGER NOT NULL DEFAULT 0,
    "min_amount_minor" INTEGER,
    "max_amount_minor" INTEGER,
    "sequence" INTEGER NOT NULL DEFAULT 0,
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_driver_pay_profile_components PRIMARY KEY (id, organization_id, business_unit_id),
    FOREIGN KEY (pay_profile_id, organization_id, business_unit_id) REFERENCES driver_pay_profiles(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_pay_profile_components_profile ON driver_pay_profile_components (organization_id, business_unit_id, pay_profile_id);

--bun:split

CREATE TABLE IF NOT EXISTS worker_pay_assignments(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "pay_profile_id" TEXT NOT NULL,
    "effective_from" INTEGER NOT NULL,
    "effective_to" INTEGER,
    "split_percent" REAL NOT NULL DEFAULT 100,
    "notes" TEXT,
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_worker_pay_assignments PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_worker_pay_assignments_split CHECK (split_percent > 0 AND split_percent <= 100),
    CONSTRAINT chk_worker_pay_assignments_range CHECK (effective_to IS NULL OR effective_to > effective_from),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (pay_profile_id, organization_id, business_unit_id) REFERENCES driver_pay_profiles(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_worker_pay_assignments_worker ON worker_pay_assignments (organization_id, business_unit_id, worker_id, effective_from DESC);

--bun:split

CREATE TABLE IF NOT EXISTS escrow_accounts(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "target_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "balance_minor" INTEGER NOT NULL DEFAULT 0,
    "annual_interest_rate" REAL NOT NULL DEFAULT 0,
    "last_interest_accrual_date" INTEGER,
    "opened_date" INTEGER NOT NULL,
    "closed_date" INTEGER,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_escrow_accounts PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_escrow_accounts_target CHECK (target_amount_minor >= 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE RESTRICT
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_escrow_accounts_worker_active ON escrow_accounts (organization_id, business_unit_id, worker_id)WHERE status = 'Active';

--bun:split

CREATE TABLE IF NOT EXISTS escrow_transactions(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "escrow_account_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "balance_after_minor" INTEGER NOT NULL,
    "occurred_date" INTEGER NOT NULL,
    "description" TEXT,
    "settlement_id" TEXT,
    "created_by_id" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_escrow_transactions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_escrow_transactions_amount CHECK (amount_minor <> 0),
    FOREIGN KEY (escrow_account_id, organization_id, business_unit_id) REFERENCES escrow_accounts(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_escrow_transactions_account ON escrow_transactions (organization_id, business_unit_id, escrow_account_id, occurred_date DESC);

--bun:split

CREATE TABLE IF NOT EXISTS recurring_deductions(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "escrow_account_id" TEXT,
    "type" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "frequency" TEXT NOT NULL DEFAULT 'EverySettlement',
    "description" TEXT NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "total_cap_minor" INTEGER,
    "deducted_to_date_minor" INTEGER NOT NULL DEFAULT 0,
    "start_date" INTEGER NOT NULL,
    "end_date" INTEGER,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_recurring_deductions PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_recurring_deductions_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_recurring_deductions_cap CHECK (total_cap_minor IS NULL OR total_cap_minor > 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (escrow_account_id, organization_id, business_unit_id) REFERENCES escrow_accounts(id, organization_id, business_unit_id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_recurring_deductions_worker ON recurring_deductions (organization_id, business_unit_id, worker_id, status);

--bun:split

CREATE TABLE IF NOT EXISTS pay_advances(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Outstanding',
    "source" TEXT NOT NULL,
    "reference" TEXT,
    "issued_date" INTEGER NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "recovered_minor" INTEGER NOT NULL DEFAULT 0,
    "written_off_minor" INTEGER NOT NULL DEFAULT 0,
    "write_off_reason" TEXT,
    "notes" TEXT,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "created_by_id" TEXT,
    "written_off_by_id" TEXT,
    "written_off_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_pay_advances PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_pay_advances_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_pay_advances_recovery CHECK (recovered_minor >= 0 AND written_off_minor >= 0 AND recovered_minor + written_off_minor <= amount_minor),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (written_off_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_pay_advances_worker ON pay_advances (organization_id, business_unit_id, worker_id, status);

--bun:split

CREATE TABLE IF NOT EXISTS driver_settlement_batches(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Open',
    "name" TEXT NOT NULL,
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "pay_date" INTEGER NOT NULL,
    "settlement_count" INTEGER NOT NULL DEFAULT 0,
    "exception_count" INTEGER NOT NULL DEFAULT 0,
    "total_gross_minor" INTEGER NOT NULL DEFAULT 0,
    "total_net_minor" INTEGER NOT NULL DEFAULT 0,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "notes" TEXT,
    "generated_by_id" TEXT,
    "generated_at" INTEGER,
    "completed_at" INTEGER,
    "canceled_by_id" TEXT,
    "canceled_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_driver_settlement_batches PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_settlement_batches_period CHECK (period_end > period_start),
    FOREIGN KEY (generated_by_id) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (canceled_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_settlement_batches_status ON driver_settlement_batches (organization_id, business_unit_id, status, period_end DESC);

--bun:split

CREATE TABLE IF NOT EXISTS driver_settlements(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "batch_id" TEXT,
    "pay_profile_id" TEXT,
    "settlement_number" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "classification" TEXT NOT NULL,
    "pay_profile_name" TEXT,
    "period_start" INTEGER NOT NULL,
    "period_end" INTEGER NOT NULL,
    "pay_date" INTEGER NOT NULL,
    "gross_earnings_minor" INTEGER NOT NULL DEFAULT 0,
    "reimbursements_minor" INTEGER NOT NULL DEFAULT 0,
    "deductions_minor" INTEGER NOT NULL DEFAULT 0,
    "carry_forward_in_minor" INTEGER NOT NULL DEFAULT 0,
    "carry_forward_out_minor" INTEGER NOT NULL DEFAULT 0,
    "net_pay_minor" INTEGER NOT NULL DEFAULT 0,
    "total_miles" REAL NOT NULL DEFAULT 0,
    "shipment_count" INTEGER NOT NULL DEFAULT 0,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "has_exceptions" INTEGER NOT NULL DEFAULT 0,
    "exceptions" TEXT,
    "notes" TEXT,
    "submitted_by_id" TEXT,
    "submitted_at" INTEGER,
    "approved_by_id" TEXT,
    "approved_at" INTEGER,
    "posted_by_id" TEXT,
    "posted_at" INTEGER,
    "posted_journal_batch_id" TEXT,
    "paid_at" INTEGER,
    "paid_by_id" TEXT,
    "payment_method" TEXT,
    "payment_reference" TEXT,
    "voided_by_id" TEXT,
    "voided_at" INTEGER,
    "void_reason" TEXT,
    "void_journal_batch_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS uq_driver_settlements_number ON driver_settlements (organization_id, business_unit_id, settlement_number);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_settlements_worker ON driver_settlements (organization_id, business_unit_id, worker_id, period_end DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_settlements_status ON driver_settlements (organization_id, business_unit_id, status);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_settlements_batch ON driver_settlements (organization_id, business_unit_id, batch_id)WHERE batch_id IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS driver_settlement_lines(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "settlement_id" TEXT NOT NULL,
    "line_number" INTEGER NOT NULL,
    "category" TEXT NOT NULL,
    "component_kind" TEXT,
    "method" TEXT,
    "description" TEXT NOT NULL,
    "quantity" REAL NOT NULL DEFAULT 0,
    "rate" REAL NOT NULL DEFAULT 0,
    "amount_minor" INTEGER NOT NULL,
    "shipment_id" TEXT,
    "move_id" TEXT,
    "pay_event_id" TEXT,
    "recurring_deduction_id" TEXT,
    "advance_id" TEXT,
    "escrow_account_id" TEXT,
    "pro_number" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_driver_settlement_lines PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT uq_driver_settlement_lines_number UNIQUE (settlement_id, organization_id, business_unit_id, line_number),
    FOREIGN KEY (settlement_id, organization_id, business_unit_id) REFERENCES driver_settlements(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_settlement_lines_settlement ON driver_settlement_lines (organization_id, business_unit_id, settlement_id);

--bun:split

CREATE TABLE IF NOT EXISTS driver_pay_events(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "shipment_id" TEXT NOT NULL,
    "move_id" TEXT,
    "assignment_id" TEXT,
    "pay_profile_id" TEXT,
    "settlement_id" TEXT,
    "settlement_line_id" TEXT,
    "idempotency_key" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Accrued',
    "event_date" INTEGER NOT NULL,
    "gross_amount_minor" INTEGER NOT NULL DEFAULT 0,
    "total_miles" REAL NOT NULL DEFAULT 0,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "components" TEXT,
    "pro_number" TEXT,
    "voided_at" INTEGER,
    "void_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_driver_pay_events PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_pay_events_gross CHECK (gross_amount_minor >= 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (shipment_id, organization_id, business_unit_id) REFERENCES shipments(id, organization_id, business_unit_id) ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_driver_pay_events_idempotency ON driver_pay_events (organization_id, business_unit_id, idempotency_key);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_pay_events_worker ON driver_pay_events (organization_id, business_unit_id, worker_id, status, event_date DESC);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_pay_events_shipment ON driver_pay_events (organization_id, business_unit_id, shipment_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_pay_events_settlement ON driver_pay_events (organization_id, business_unit_id, settlement_id)WHERE settlement_id IS NOT NULL;
