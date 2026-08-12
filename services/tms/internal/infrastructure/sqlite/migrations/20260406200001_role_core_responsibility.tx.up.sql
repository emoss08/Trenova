-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260406200001_role_core_responsibility.tx.up.sql

ALTER TABLE "roles" ADD COLUMN "core_responsibility" TEXT;
