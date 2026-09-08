-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261025000000_position_holders.tx.down.sql

DROP INDEX IF EXISTS "idx_user_organization_memberships_position";

--bun:split
ALTER TABLE "user_organization_memberships"
    DROP COLUMN "position_id";
