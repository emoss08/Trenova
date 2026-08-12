-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260802000000_pay_codes.tx.up.sql

CREATE TABLE IF NOT EXISTS pay_codes(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "direction" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "taxable" INTEGER NOT NULL DEFAULT 1,
    "counts_toward_guarantee" INTEGER NOT NULL DEFAULT 1,
    "gl_account_id" TEXT,
    "default_amount_minor" INTEGER,
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT pk_pay_codes PRIMARY KEY (id, organization_id, business_unit_id),
    CONSTRAINT chk_pay_codes_direction CHECK (direction IN ('Earning', 'Deduction')),
    CONSTRAINT chk_pay_codes_default_amount CHECK (default_amount_minor IS NULL OR default_amount_minor > 0),
    FOREIGN KEY (gl_account_id, organization_id, business_unit_id) REFERENCES gl_accounts(id, organization_id, business_unit_id) ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS uq_pay_codes_org_direction_code ON pay_codes (organization_id, business_unit_id, direction, code);

--bun:split

ALTER TABLE "recurring_earnings" ADD COLUMN "pay_code_id" TEXT;

--bun:split

ALTER TABLE "recurring_deductions" ADD COLUMN "pay_code_id" TEXT;

--bun:split

ALTER TABLE "driver_settlement_lines" ADD COLUMN "pay_code_id" TEXT;
