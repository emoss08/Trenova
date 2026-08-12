-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260320123000_add_is_system_generated_to_additional_charges.tx.up.sql

ALTER TABLE "additional_charges" ADD COLUMN "is_system_generated" INTEGER NOT NULL DEFAULT 0;
