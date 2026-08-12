-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260805000000_driver_expenses.tx.up.sql

CREATE TABLE IF NOT EXISTS driver_expenses(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "worker_id" TEXT NOT NULL,
    "shipment_id" TEXT,
    "pay_code_id" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "amount_minor" INTEGER NOT NULL,
    "currency_code" TEXT NOT NULL DEFAULT 'USD',
    "description" TEXT NOT NULL,
    "incurred_date" INTEGER NOT NULL,
    "receipt_document_id" TEXT,
    "submitted_by_user_id" TEXT NOT NULL,
    "review_note" TEXT,
    "reviewed_by_id" TEXT,
    "reviewed_at" INTEGER,
    "settlement_line_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_driver_expenses PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_driver_expenses_status CHECK (status IN ('Pending', 'Approved', 'Rejected', 'Reimbursed', 'Cancelled')),
    CONSTRAINT chk_driver_expenses_amount CHECK (amount_minor > 0),
    FOREIGN KEY (worker_id, organization_id, business_unit_id) REFERENCES workers(id, organization_id, business_unit_id) ON DELETE CASCADE,
    FOREIGN KEY (pay_code_id, organization_id, business_unit_id) REFERENCES pay_codes(id, organization_id, business_unit_id) ON DELETE SET NULL,
    FOREIGN KEY (submitted_by_user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (reviewed_by_id) REFERENCES users(id) ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_expenses_worker ON driver_expenses (worker_id, organization_id, business_unit_id);

--bun:split

CREATE INDEX IF NOT EXISTS idx_driver_expenses_status ON driver_expenses (organization_id, business_unit_id, status)WHERE
    status = 'Pending';

--bun:split

ALTER TABLE "assignments" ADD COLUMN "ack_status" TEXT NOT NULL DEFAULT 'Pending';

--bun:split

ALTER TABLE "assignments" ADD COLUMN "ack_at" INTEGER;

--bun:split

ALTER TABLE "assignments" ADD COLUMN "ack_reason" TEXT;
