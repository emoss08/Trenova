ALTER TABLE "carrier_intel_snapshots"
    DROP CONSTRAINT IF EXISTS "chk_carrier_intel_snapshots_fetched_depth";

--bun:split
ALTER TABLE "carrier_intel_snapshots"
    DROP COLUMN IF EXISTS "depth_fetched_at",
    DROP COLUMN IF EXISTS "fetched_depth";
