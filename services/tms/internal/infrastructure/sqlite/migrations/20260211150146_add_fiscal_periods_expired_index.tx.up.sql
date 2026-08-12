-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260211150146_add_fiscal_periods_expired_index.tx.up.sql

CREATE INDEX IF NOT EXISTS idx_fiscal_periods_expired_open
    ON "fiscal_periods" ("organization_id", "business_unit_id", "status", "end_date")WHERE "status" = 'Open';
