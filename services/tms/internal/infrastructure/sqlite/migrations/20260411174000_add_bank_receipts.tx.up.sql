-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260411174000_add_bank_receipts.tx.up.sql

CREATE TABLE IF NOT EXISTS bank_receipts(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "receipt_date" INTEGER NOT NULL,
    "amount_minor" INTEGER NOT NULL,
    "reference_number" TEXT,
    "memo" TEXT,
    "status" TEXT NOT NULL,
    "matched_customer_payment_id" TEXT,
    "matched_at" INTEGER,
    "matched_by_id" TEXT,
    "exception_reason" TEXT,
    "created_by_id" TEXT NOT NULL,
    "updated_by_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_bank_receipts PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT fk_bank_receipts_payment FOREIGN KEY (matched_customer_payment_id, organization_id, business_unit_id) REFERENCES customer_payments(id, organization_id, business_unit_id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipts_created_by FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT fk_bank_receipts_updated_by FOREIGN KEY (updated_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT fk_bank_receipts_matched_by FOREIGN KEY (matched_by_id) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_bank_receipts_amount CHECK (amount_minor > 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_bank_receipts_status_date ON bank_receipts (organization_id, business_unit_id, status, receipt_date);
