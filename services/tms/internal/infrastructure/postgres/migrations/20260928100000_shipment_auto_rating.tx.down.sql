DROP INDEX IF EXISTS "idx_shipments_auto_rated";

--bun:split
ALTER TABLE "shipments" DROP COLUMN IF EXISTS "auto_rated_at";

--bun:split
ALTER TABLE "shipments" DROP COLUMN IF EXISTS "auto_rated";
