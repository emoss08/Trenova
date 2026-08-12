-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260827000000_home_layout_index_cleanup.tx.up.sql

DROP INDEX IF EXISTS "idx_home_layout_presets_role_ids";

--bun:split

DROP INDEX IF EXISTS "idx_home_layout_presets_tenant";
