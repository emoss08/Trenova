-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260927100000_merge_rate_tables_into_matrices.tx.up.sql

DROP TABLE IF EXISTS "rate_table_entries";

--bun:split

DROP TABLE IF EXISTS "rate_tables";
