-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260801000000_recurring_earnings.tx.up.sql

CREATE TABLE IF NOT EXISTS recurring_earnings(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "frequency" TEXT NOT NULL DEFAULT 'EverySettlement',
    "description" TEXT NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "total_cap_minor" INTEGER,
    "paid_to_date_minor" INTEGER NOT NULL DEFAULT 0,
    "start_date" INTEGER NOT NULL,
    "end_date" INTEGER,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "created_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_recurring_earnings PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_recurring_earnings_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_recurring_earnings_cap CHECK (total_cap_minor IS NULL OR total_cap_minor > 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_recurring_earnings_worker ON recurring_earnings (organization_id, business_unit_id, worker_id, status);

--bun:split

ALTER TABLE "driver_settlement_lines" ADD COLUMN "recurring_earning_id" TEXT;

--bun:split

ALTER TABLE "accounting_controls" ADD COLUMN "default_driver_reimbursement_account_id" TEXT;
