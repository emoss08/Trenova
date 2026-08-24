-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260915000000_organization_capability_flags.tx.up.sql

ALTER TABLE "organizations" ADD COLUMN "brokerage_enabled" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "organizations" ADD COLUMN "asset_operations_enabled" INTEGER NOT NULL DEFAULT 1;
