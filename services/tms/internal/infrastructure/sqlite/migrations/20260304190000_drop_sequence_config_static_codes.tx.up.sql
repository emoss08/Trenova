-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260304190000_drop_sequence_config_static_codes.tx.up.sql

ALTER TABLE "sequence_configs" DROP COLUMN "location_code";

--bun:split

ALTER TABLE "sequence_configs" DROP COLUMN "business_unit_code";
