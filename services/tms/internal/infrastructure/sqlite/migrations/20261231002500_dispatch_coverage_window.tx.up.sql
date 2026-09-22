-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231002500_dispatch_coverage_window.tx.up.sql

ALTER TABLE "dispatch_controls" ADD COLUMN "coverage_risk_window_hours" INTEGER NOT NULL DEFAULT 12;
