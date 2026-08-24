-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260923000000_routing_guide_contract_rate.tx.up.sql

ALTER TABLE "routing_guide_entries" ADD COLUMN "use_contract_rate" INTEGER NOT NULL DEFAULT 0;
