-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261230000300_carrier_intel_snapshot_depth_freshness.tx.up.sql

ALTER TABLE "carrier_intel_snapshots" ADD COLUMN "fetched_depth" TEXT;

--bun:split

ALTER TABLE "carrier_intel_snapshots" ADD COLUMN "depth_fetched_at" INTEGER;

--bun:split

UPDATE
    "carrier_intel_snapshots"
SET
    "fetched_depth" = COALESCE("fetched_depth", "depth"),
    "depth_fetched_at" = COALESCE("depth_fetched_at", "fetched_at")
WHERE
    "fetched_depth" IS NULL
    OR "depth_fetched_at" IS NULL;
