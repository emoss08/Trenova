-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260917000000_dispatch_horizon_search.tx.up.sql

ALTER TABLE "dispatch_controls" ADD COLUMN "horizon_search_iterations" INTEGER;
