-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260420120000_add_location_geofences.tx.up.sql

ALTER TABLE "locations" ADD COLUMN "geofence_type" TEXT NOT NULL DEFAULT 'auto';

--bun:split

ALTER TABLE "locations" ADD COLUMN "geofence_radius_meters" REAL;
