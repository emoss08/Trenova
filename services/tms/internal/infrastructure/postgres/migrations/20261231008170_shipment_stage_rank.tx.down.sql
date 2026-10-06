DROP INDEX IF EXISTS "idx_shipments_stage_rank";

--bun:split
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "stage_rank";
