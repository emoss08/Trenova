-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260918000000_dispatch_planning_mode.tx.up.sql

ALTER TABLE "dispatch_controls" ADD COLUMN "planning_mode" TEXT NOT NULL DEFAULT 'Immediate';

--bun:split

ALTER TABLE "dispatch_controls" ADD COLUMN "horizon_max_moves_per_driver" INTEGER NOT NULL DEFAULT 3;
