-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260405120000_add_equipment_type_interior_length.tx.up.sql

ALTER TABLE "equipment_types" ADD COLUMN "interior_length" NUMERIC;
