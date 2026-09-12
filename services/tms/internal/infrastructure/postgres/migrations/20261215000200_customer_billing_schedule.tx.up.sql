--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
-- Six columns described how a customer gets billed and none of them were read by
-- any code. Worse, they overlapped: billing_cycle_type and consolidation_period_days
-- both encoded cadence and could contradict each other, invoice_method said
-- "Summary" to mean consolidation a second time, and consolidation_group_by
-- conflated how many invoices a period yields with how one invoice is organised.
--
-- They are replaced by one model on three axes — delivery mode, cadence, and the
-- split/section/detail of the result — plus the timezone the period boundaries are
-- evaluated in, which was missing entirely.
CREATE TYPE "customer_invoice_delivery_enum" AS ENUM(
    'PerShipment',
    'PerOrder',
    'Consolidated'
);

--bun:split
CREATE TYPE "customer_billing_cycle_enum" AS ENUM(
    'Immediate',
    'Daily',
    'Weekly',
    'BiWeekly',
    'SemiMonthly',
    'Monthly',
    'Quarterly'
);

--bun:split
-- No Division member: no division entity exists in the domain, and a shipment
-- carries no fleet code to stand in for one. The legacy value degrades to
-- Customer below, which is what it actually did.
CREATE TYPE "invoice_split_key_enum" AS ENUM(
    'Customer',
    'CustomerAndPONumber',
    'CustomerAndShipmentBOL',
    'CustomerAndOrder',
    'CustomerAndOrigin',
    'CustomerAndDestination',
    'CustomerAndServiceType'
);

--bun:split
CREATE TYPE "invoice_section_key_enum" AS ENUM(
    'Shipment',
    'PONumber',
    'Origin',
    'Destination'
);

--bun:split
CREATE TYPE "invoice_detail_enum" AS ENUM(
    'Detailed',
    'Summary'
);

--bun:split
-- The old columns are dropped below and their enum types with them, so the
-- down-migration cannot reconstruct them from the new model alone: Division and
-- Customer both map to Customer, and SemiMonthly has no legacy equivalent. Keep a
-- verbatim snapshot so a rollback restores what was actually configured.
CREATE TABLE "customer_billing_profiles_legacy_dials"(
    "billing_profile_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "billing_cycle_type" text,
    "billing_cycle_day_of_week" smallint,
    "invoice_method" text,
    "allow_invoice_consolidation" boolean,
    "consolidation_period_days" integer,
    "consolidation_group_by" text,
    "captured_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    CONSTRAINT "pk_customer_billing_profiles_legacy_dials" PRIMARY KEY ("billing_profile_id", "organization_id", "business_unit_id")
);

--bun:split
INSERT INTO "customer_billing_profiles_legacy_dials"("billing_profile_id", "organization_id", "business_unit_id", "billing_cycle_type", "billing_cycle_day_of_week", "invoice_method", "allow_invoice_consolidation", "consolidation_period_days", "consolidation_group_by")
SELECT
    "id",
    "organization_id",
    "business_unit_id",
    "billing_cycle_type"::text,
    "billing_cycle_day_of_week",
    "invoice_method"::text,
    "allow_invoice_consolidation",
    "consolidation_period_days",
    "consolidation_group_by"::text
FROM
    "customer_billing_profiles";

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "invoice_delivery" customer_invoice_delivery_enum NOT NULL DEFAULT 'PerShipment',
    ADD COLUMN IF NOT EXISTS "billing_cycle" customer_billing_cycle_enum NOT NULL DEFAULT 'Immediate',
    ADD COLUMN IF NOT EXISTS "billing_cycle_anchor_day" smallint NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS "billing_cycle_timezone" varchar(64) NOT NULL DEFAULT 'UTC',
    ADD COLUMN IF NOT EXISTS "last_billed_period_end" bigint,
    ADD COLUMN IF NOT EXISTS "split_by" invoice_split_key_enum NOT NULL DEFAULT 'Customer',
    ADD COLUMN IF NOT EXISTS "section_by" invoice_section_key_enum NOT NULL DEFAULT 'Shipment',
    ADD COLUMN IF NOT EXISTS "invoice_detail" invoice_detail_enum NOT NULL DEFAULT 'Detailed',
    ADD COLUMN IF NOT EXISTS "consolidation_lookback_days" smallint NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS "min_consolidated_amount" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "min_consolidated_amount_minor" bigint,
    ADD COLUMN IF NOT EXISTS "max_shipments_per_invoice" smallint NOT NULL DEFAULT 0;

--bun:split
-- Delivery mode: the consolidation flag wins over invoice_method because it is
-- the one an operator had to deliberately turn on.
UPDATE
    "customer_billing_profiles"
SET
    "invoice_delivery" = CASE WHEN "allow_invoice_consolidation" THEN
        'Consolidated'
    WHEN "invoice_method" IN ('Summary', 'SummaryWithDetail') THEN
        'Consolidated'
    ELSE
        'PerShipment'
    END::customer_invoice_delivery_enum;

--bun:split
-- Cadence: billing_cycle_type is authoritative where it names one, except that
-- PerShipment is a delivery mode rather than a cadence. Where it does not, a
-- consolidated customer's old period-days number is the only signal left.
UPDATE
    "customer_billing_profiles"
SET
    "billing_cycle" = CASE WHEN "billing_cycle_type" IN ('Daily', 'Weekly', 'BiWeekly', 'Monthly', 'Quarterly') THEN
        "billing_cycle_type"::text::customer_billing_cycle_enum
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
    END::customer_billing_cycle_enum;

--bun:split
UPDATE
    "customer_billing_profiles" cbp
SET
    "billing_cycle_anchor_day" = CASE WHEN cbp."billing_cycle" IN ('Weekly', 'BiWeekly') THEN
        COALESCE(cbp."billing_cycle_day_of_week", 1)
    ELSE
        1
    END,
    "consolidation_lookback_days" = GREATEST(COALESCE(cbp."consolidation_period_days", 30)::smallint, 7::smallint),
    "invoice_detail" = CASE WHEN cbp."invoice_method" = 'Summary' THEN
        'Summary'
    ELSE
        'Detailed'
    END::invoice_detail_enum,
    "split_by" = CASE cbp."consolidation_group_by"::text
    WHEN 'PONumber' THEN
        'CustomerAndPONumber'
    WHEN 'BOL' THEN
        'CustomerAndShipmentBOL'
    WHEN 'Location' THEN
        'CustomerAndDestination'
    ELSE
        'Customer'
    END::invoice_split_key_enum,
    "section_by" = CASE cbp."consolidation_group_by"::text
    WHEN 'PONumber' THEN
        'PONumber'
    WHEN 'Location' THEN
        'Destination'
    ELSE
        'Shipment'
    END::invoice_section_key_enum,
    "billing_cycle_timezone" = COALESCE((
        SELECT
            o."timezone"
        FROM "organizations" o
        WHERE
            o."id" = cbp."organization_id"), 'UTC');

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP CONSTRAINT IF EXISTS "customer_billing_profiles_consolidation_period_days_check";

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "billing_cycle_type",
    DROP COLUMN IF EXISTS "billing_cycle_day_of_week",
    DROP COLUMN IF EXISTS "invoice_method",
    DROP COLUMN IF EXISTS "allow_invoice_consolidation",
    DROP COLUMN IF EXISTS "consolidation_period_days",
    DROP COLUMN IF EXISTS "consolidation_group_by";

--bun:split
DROP TYPE IF EXISTS "billing_cycle_type_enum";

--bun:split
DROP TYPE IF EXISTS "invoice_method_enum";

--bun:split
DROP TYPE IF EXISTS "consolidation_group_by_enum";

--bun:split
-- The scheduled sweep asks "which profiles are periodic and overdue", so the
-- index leads with the two columns that answer it.
CREATE INDEX IF NOT EXISTS idx_customer_billing_profiles_cycle ON customer_billing_profiles("billing_cycle", "last_billed_period_end")
WHERE
    "invoice_delivery" = 'Consolidated';
