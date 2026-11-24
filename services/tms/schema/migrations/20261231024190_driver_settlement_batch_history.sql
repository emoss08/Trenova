--keel:notx
-- A pay batch keeps its reimbursements beside its gross and net, so what came off earnings
-- (gross plus reimbursements, less net) is read from the batch's stored totals.
ALTER TABLE "driver_settlement_batches"
    ADD COLUMN IF NOT EXISTS "total_reimbursements_minor" bigint NOT NULL DEFAULT 0;

--keel:split
UPDATE
    "driver_settlement_batches" AS b
SET
    "total_reimbursements_minor" = totals.reimbursements
FROM (
    SELECT
        s."batch_id",
        s."organization_id",
        s."business_unit_id",
        coalesce(sum(s."reimbursements_minor") FILTER (WHERE s."status" <> 'Voided'), 0) AS reimbursements
    FROM
        "driver_settlements" AS s
    WHERE
        s."batch_id" IS NOT NULL
    GROUP BY
        s."batch_id",
        s."organization_id",
        s."business_unit_id") AS totals
WHERE
    totals."batch_id" = b."id"
    AND totals."organization_id" = b."organization_id"
    AND totals."business_unit_id" = b."business_unit_id"
    AND b."total_reimbursements_minor" IS DISTINCT FROM totals.reimbursements;

--keel:split
-- The newest pay periods first, for the batch a board reads and the five before it.
CREATE INDEX CONCURRENTLY IF NOT EXISTS "idx_driver_settlement_batches_period" ON "driver_settlement_batches"("organization_id", "business_unit_id", "period_start" DESC);
