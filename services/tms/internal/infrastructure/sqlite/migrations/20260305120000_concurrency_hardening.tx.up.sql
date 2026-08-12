-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260305120000_concurrency_hardening.tx.up.sql

DROP INDEX IF EXISTS idx_fiscal_years_unique_current;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_fiscal_years_unique_current
    ON fiscal_years (organization_id, business_unit_id)WHERE is_current = TRUE;

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS idx_user_org_memberships_one_default_per_bu
    ON user_organization_memberships (user_id, business_unit_id)WHERE is_default = TRUE;
