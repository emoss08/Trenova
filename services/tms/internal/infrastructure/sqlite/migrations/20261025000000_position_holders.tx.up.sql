-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261025000000_position_holders.tx.up.sql

ALTER TABLE "user_organization_memberships" ADD COLUMN "position_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_user_organization_memberships_position" ON "user_organization_memberships" ("organization_id", "business_unit_id", "position_id")WHERE
    "position_id" IS NOT NULL;
