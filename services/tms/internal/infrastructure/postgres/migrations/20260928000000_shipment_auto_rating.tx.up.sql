-- A contract rate is now applied to a shipment once, into the shipment's own
-- rating fields, rather than re-resolved on every recalculation. These columns
-- are what separates a shipment still carrying the contract's answer from one
-- somebody has since priced by hand, which is what the contract usage and rate
-- leakage reports are counted from.
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "auto_rated" boolean NOT NULL DEFAULT false;

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "auto_rated_at" bigint;

--bun:split
-- Every shipment a contract has priced up to now was re-rated on each save, so
-- the ones carrying an agreement stamp and no hand-set override are exactly the
-- ones the new model would call auto-rated. Backfilling them keeps their badge
-- and their exclusion from re-rating correct from the first deploy.
UPDATE "shipments"
SET "auto_rated" = true,
    "auto_rated_at" = "updated_at"
WHERE "rate_agreement_id" IS NOT NULL
    AND "rate_override_amount" IS NULL;

--bun:split
-- The usage report reads this in the same breath as the agreement stamp.
CREATE INDEX IF NOT EXISTS "idx_shipments_auto_rated"
    ON "shipments" ("organization_id", "auto_rated")
    WHERE "auto_rated";
