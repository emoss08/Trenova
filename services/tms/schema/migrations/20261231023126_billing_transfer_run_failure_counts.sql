-- A transfer run keeps how many of its shipments each failure reason held back,
-- beside the outcome counters it already derives from its items. The dialog reads
-- it to group the shipments that did not transfer by the work left to do, while
-- the run is going and once it ends, without counting the items on every poll.
ALTER TABLE "billing_transfer_runs"
    ADD COLUMN IF NOT EXISTS "failure_counts" jsonb NOT NULL DEFAULT '[]';

UPDATE "billing_transfer_runs" AS "run"
SET "failure_counts" = "counted"."failure_counts"
FROM (
    SELECT
        "grouped"."run_id",
        "grouped"."business_unit_id",
        "grouped"."organization_id",
        jsonb_agg(
            jsonb_build_object('code', "grouped"."failure_code", 'count', "grouped"."total")
            ORDER BY "grouped"."total" DESC, "grouped"."failure_code"
        ) AS "failure_counts"
    FROM (
        SELECT "run_id", "business_unit_id", "organization_id", "failure_code", count(*) AS "total"
        FROM "billing_transfer_run_items"
        WHERE "status" = 'NotTransferred'
        GROUP BY "run_id", "business_unit_id", "organization_id", "failure_code"
    ) AS "grouped"
    GROUP BY "grouped"."run_id", "grouped"."business_unit_id", "grouped"."organization_id"
) AS "counted"
WHERE "run"."id" = "counted"."run_id"
    AND "run"."business_unit_id" = "counted"."business_unit_id"
    AND "run"."organization_id" = "counted"."organization_id";

COMMENT ON COLUMN "billing_transfer_runs"."failure_counts" IS 'How many shipments each failure reason held back, as [{code, count}] ordered by count, refreshed with the other counters after every batch.';

-- The report filters the shipments that did not transfer by their reason.
CREATE INDEX IF NOT EXISTS "idx_billing_transfer_run_items_run_failure"
    ON "billing_transfer_run_items" ("organization_id", "business_unit_id", "run_id", "failure_code")
    WHERE "status" = 'NotTransferred';
