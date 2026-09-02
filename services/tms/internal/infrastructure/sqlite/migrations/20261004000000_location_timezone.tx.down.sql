-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261004000000_location_timezone.tx.down.sql
ALTER TABLE "locations"
DROP COLUMN "timezone";
