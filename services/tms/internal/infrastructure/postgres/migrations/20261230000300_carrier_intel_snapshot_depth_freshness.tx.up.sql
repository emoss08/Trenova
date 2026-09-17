ALTER TABLE "carrier_intel_snapshots"
    ADD COLUMN IF NOT EXISTS "fetched_depth" varchar(10),
    ADD COLUMN IF NOT EXISTS "depth_fetched_at" bigint;

--bun:split
UPDATE
    "carrier_intel_snapshots"
SET
    "fetched_depth" = COALESCE("fetched_depth", "depth"),
    "depth_fetched_at" = COALESCE("depth_fetched_at", "fetched_at")
WHERE
    "fetched_depth" IS NULL
    OR "depth_fetched_at" IS NULL;

--bun:split
ALTER TABLE "carrier_intel_snapshots"
    ALTER COLUMN "fetched_depth" SET NOT NULL,
    ALTER COLUMN "depth_fetched_at" SET NOT NULL;

--bun:split
ALTER TABLE "carrier_intel_snapshots"
    DROP CONSTRAINT IF EXISTS "chk_carrier_intel_snapshots_fetched_depth";

--bun:split
ALTER TABLE "carrier_intel_snapshots"
    ADD CONSTRAINT "chk_carrier_intel_snapshots_fetched_depth" CHECK ("fetched_depth" IN ('FMCSA', 'Lite', 'Full'));

--bun:split
COMMENT ON COLUMN "carrier_intel_snapshots"."fetched_depth" IS 'Depth of the provider response that produced or last confirmed this snapshot. Lower than depth when a shallower fetch was merged over a deeper profile still within its TTL.';

--bun:split
COMMENT ON COLUMN "carrier_intel_snapshots"."depth_fetched_at" IS 'When data at the snapshot''s depth was last fetched from the provider. Freshness checks for that depth use this instead of fetched_at so a shallower merge never extends a deeper profile''s TTL.';
