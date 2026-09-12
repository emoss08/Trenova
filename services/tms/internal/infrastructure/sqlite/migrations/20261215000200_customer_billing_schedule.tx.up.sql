-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261215000200_customer_billing_schedule.tx.up.sql
--
-- Hand-completed: the converter drops the snapshot INSERT and the three backfill
-- UPDATEs. billing_cycle_day_of_week and consolidation_period_days are also left
-- in place, because SQLite refuses to drop a column a CHECK constraint still
-- names and will not rebuild the table to do it. Both are simply unused here,
-- since the Go struct no longer declares them.
CREATE TABLE IF NOT EXISTS "customer_billing_profiles_legacy_dials"(
    "billing_profile_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "billing_cycle_type" TEXT,
    "billing_cycle_day_of_week" INTEGER,
    "invoice_method" TEXT,
    "allow_invoice_consolidation" INTEGER,
    "consolidation_period_days" INTEGER,
    "consolidation_group_by" TEXT,
    "captured_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_customer_billing_profiles_legacy_dials" PRIMARY KEY ("billing_profile_id", "organization_id", "business_unit_id")
);

--bun:split

INSERT INTO "customer_billing_profiles_legacy_dials"("billing_profile_id", "organization_id", "business_unit_id", "billing_cycle_type", "billing_cycle_day_of_week", "invoice_method", "allow_invoice_consolidation", "consolidation_period_days", "consolidation_group_by")
SELECT
    "id",
    "organization_id",
    "business_unit_id",
    "billing_cycle_type",
    "billing_cycle_day_of_week",
    "invoice_method",
    "allow_invoice_consolidation",
    "consolidation_period_days",
    "consolidation_group_by"
FROM
    "customer_billing_profiles";

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "invoice_delivery" TEXT NOT NULL DEFAULT 'PerShipment';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "billing_cycle" TEXT NOT NULL DEFAULT 'Immediate';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "billing_cycle_anchor_day" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "billing_cycle_timezone" TEXT NOT NULL DEFAULT 'UTC';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "last_billed_period_end" INTEGER;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "split_by" TEXT NOT NULL DEFAULT 'Customer';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "section_by" TEXT NOT NULL DEFAULT 'Shipment';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "invoice_detail" TEXT NOT NULL DEFAULT 'Detailed';

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "consolidation_lookback_days" INTEGER NOT NULL DEFAULT 30;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "min_consolidated_amount" REAL;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "min_consolidated_amount_minor" INTEGER;

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "max_shipments_per_invoice" INTEGER NOT NULL DEFAULT 0;

--bun:split

UPDATE
    "customer_billing_profiles"
SET
    "invoice_delivery" = CASE WHEN "allow_invoice_consolidation" = 1 THEN
        'Consolidated'
    WHEN "invoice_method" IN ('Summary', 'SummaryWithDetail') THEN
        'Consolidated'
    ELSE
        'PerShipment'
    END;

--bun:split

UPDATE
    "customer_billing_profiles"
SET
    "billing_cycle" = CASE WHEN "billing_cycle_type" IN ('Daily', 'Weekly', 'BiWeekly', 'Monthly', 'Quarterly') THEN
        "billing_cycle_type"
    WHEN "invoice_delivery" = 'Consolidated'
        AND "consolidation_period_days" >= 28 THEN
        'Monthly'
    WHEN "invoice_delivery" = 'Consolidated'
        AND "consolidation_period_days" >= 14 THEN
        'BiWeekly'
    WHEN "invoice_delivery" = 'Consolidated'
        AND "consolidation_period_days" >= 7 THEN
        'Weekly'
    WHEN "invoice_delivery" = 'Consolidated' THEN
        'Daily'
    ELSE
        'Immediate'
    END;

--bun:split

UPDATE
    "customer_billing_profiles"
SET
    "billing_cycle_anchor_day" = CASE WHEN "billing_cycle" IN ('Weekly', 'BiWeekly') THEN
        COALESCE("billing_cycle_day_of_week", 1)
    ELSE
        1
    END,
    "consolidation_lookback_days" = MAX(COALESCE("consolidation_period_days", 30), 7),
    "invoice_detail" = CASE WHEN "invoice_method" = 'Summary' THEN
        'Summary'
    ELSE
        'Detailed'
    END,
    "split_by" = CASE "consolidation_group_by"
    WHEN 'PONumber' THEN
        'CustomerAndPONumber'
    WHEN 'BOL' THEN
        'CustomerAndShipmentBOL'
    WHEN 'Location' THEN
        'CustomerAndDestination'
    ELSE
        'Customer'
    END,
    "section_by" = CASE "consolidation_group_by"
    WHEN 'PONumber' THEN
        'PONumber'
    WHEN 'Location' THEN
        'Destination'
    ELSE
        'Shipment'
    END,
    "billing_cycle_timezone" = COALESCE((
        SELECT
            o."timezone"
        FROM "organizations" o
        WHERE
            o."id" = "customer_billing_profiles"."organization_id"), 'UTC');

--bun:split

DROP INDEX IF EXISTS "idx_customer_billing_profiles_billing_cycle";

--bun:split

ALTER TABLE "customer_billing_profiles" DROP COLUMN "billing_cycle_type";

--bun:split

ALTER TABLE "customer_billing_profiles" DROP COLUMN "invoice_method";

--bun:split

ALTER TABLE "customer_billing_profiles" DROP COLUMN "allow_invoice_consolidation";

--bun:split

ALTER TABLE "customer_billing_profiles" DROP COLUMN "consolidation_group_by";

--bun:split

CREATE INDEX IF NOT EXISTS idx_customer_billing_profiles_cycle ON customer_billing_profiles ("billing_cycle", "last_billed_period_end")
WHERE
    "invoice_delivery" = 'Consolidated';
