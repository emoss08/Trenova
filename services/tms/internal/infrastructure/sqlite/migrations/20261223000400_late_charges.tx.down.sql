-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py, then hand-completed: SQLite cannot drop
-- the billing control columns, so only the table is undone. Hand-edits are
-- preserved only if you stop regenerating this file; see docs/databases.md.
-- Source: 20261223000400_late_charges.tx.down.sql

DROP TABLE IF EXISTS "late_charge_assessments";
