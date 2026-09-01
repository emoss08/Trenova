-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261003000000_rate_matrix_dimension_modes.tx.down.sql
ALTER TABLE "rate_matrix_dimensions"
DROP COLUMN "key_normalization";

--bun:split
ALTER TABLE "rate_matrix_dimensions"
DROP COLUMN "range_overflow";
