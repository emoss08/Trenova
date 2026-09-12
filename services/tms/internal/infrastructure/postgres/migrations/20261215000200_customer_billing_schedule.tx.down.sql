--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
DROP INDEX IF EXISTS idx_customer_billing_profiles_cycle;

--bun:split
CREATE TYPE "billing_cycle_type_enum" AS ENUM(
    'Immediate',
    'Daily',
    'Weekly',
    'BiWeekly',
    'Monthly',
    'Quarterly',
    'PerShipment'
);

--bun:split
CREATE TYPE "invoice_method_enum" AS ENUM(
    'Individual',
    'Summary',
    'SummaryWithDetail'
);

--bun:split
CREATE TYPE "consolidation_group_by_enum" AS ENUM(
    'None',
    'Location',
    'PONumber',
    'BOL',
    'Division'
);

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "billing_cycle_type" billing_cycle_type_enum DEFAULT 'Immediate',
    ADD COLUMN IF NOT EXISTS "billing_cycle_day_of_week" smallint,
    ADD COLUMN IF NOT EXISTS "invoice_method" invoice_method_enum NOT NULL DEFAULT 'Individual',
    ADD COLUMN IF NOT EXISTS "allow_invoice_consolidation" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "consolidation_period_days" integer NOT NULL DEFAULT 7,
    ADD COLUMN IF NOT EXISTS "consolidation_group_by" consolidation_group_by_enum NOT NULL DEFAULT 'None';

--bun:split
-- Restore verbatim from the snapshot rather than deriving from the new model:
-- Division and Customer both mapped to Customer on the way up, and SemiMonthly
-- has no legacy equivalent, so derivation would silently change configuration.
UPDATE
    "customer_billing_profiles" cbp
SET
    "billing_cycle_type" = legacy."billing_cycle_type"::billing_cycle_type_enum,
    "billing_cycle_day_of_week" = legacy."billing_cycle_day_of_week",
    "invoice_method" = COALESCE(legacy."invoice_method"::invoice_method_enum, 'Individual'),
    "allow_invoice_consolidation" = COALESCE(legacy."allow_invoice_consolidation", FALSE),
    "consolidation_period_days" = COALESCE(legacy."consolidation_period_days", 7),
    "consolidation_group_by" = COALESCE(legacy."consolidation_group_by"::consolidation_group_by_enum, 'None')
FROM
    "customer_billing_profiles_legacy_dials" legacy
WHERE
    legacy."billing_profile_id" = cbp."id"
    AND legacy."organization_id" = cbp."organization_id"
    AND legacy."business_unit_id" = cbp."business_unit_id";

--bun:split
-- Profiles created after the up-migration have no snapshot row; give them the
-- closest legacy equivalent of their current configuration.
UPDATE
    "customer_billing_profiles" cbp
SET
    "allow_invoice_consolidation" = (cbp."invoice_delivery" = 'Consolidated'),
    "billing_cycle_type" = CASE WHEN cbp."billing_cycle" = 'SemiMonthly' THEN
        'BiWeekly'
    ELSE
        cbp."billing_cycle"::text::billing_cycle_type_enum
    END,
    "consolidation_group_by" = CASE cbp."split_by"::text
    WHEN 'CustomerAndPONumber' THEN
        'PONumber'
    WHEN 'CustomerAndShipmentBOL' THEN
        'BOL'
    WHEN 'CustomerAndOrigin' THEN
        'Location'
    WHEN 'CustomerAndDestination' THEN
        'Location'
    ELSE
        'None'
    END::consolidation_group_by_enum,
    "invoice_method" = CASE WHEN cbp."invoice_delivery" <> 'Consolidated' THEN
        'Individual'
    WHEN cbp."invoice_detail" = 'Summary' THEN
        'Summary'
    ELSE
        'SummaryWithDetail'
    END::invoice_method_enum,
    "consolidation_period_days" = GREATEST(cbp."consolidation_lookback_days", 1)
WHERE
    NOT EXISTS (
        SELECT
            1
        FROM "customer_billing_profiles_legacy_dials" legacy
        WHERE
            legacy."billing_profile_id" = cbp."id"
            AND legacy."organization_id" = cbp."organization_id"
            AND legacy."business_unit_id" = cbp."business_unit_id");

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD CONSTRAINT "customer_billing_profiles_consolidation_period_days_check" CHECK ("consolidation_period_days" >= 1);

--bun:split
ALTER TABLE "customer_billing_profiles"
    DROP COLUMN IF EXISTS "invoice_delivery",
    DROP COLUMN IF EXISTS "billing_cycle",
    DROP COLUMN IF EXISTS "billing_cycle_anchor_day",
    DROP COLUMN IF EXISTS "billing_cycle_timezone",
    DROP COLUMN IF EXISTS "last_billed_period_end",
    DROP COLUMN IF EXISTS "split_by",
    DROP COLUMN IF EXISTS "section_by",
    DROP COLUMN IF EXISTS "invoice_detail",
    DROP COLUMN IF EXISTS "consolidation_lookback_days",
    DROP COLUMN IF EXISTS "min_consolidated_amount",
    DROP COLUMN IF EXISTS "min_consolidated_amount_minor",
    DROP COLUMN IF EXISTS "max_shipments_per_invoice";

--bun:split
DROP TABLE IF EXISTS "customer_billing_profiles_legacy_dials";

--bun:split
DROP TYPE IF EXISTS "customer_invoice_delivery_enum";

--bun:split
DROP TYPE IF EXISTS "customer_billing_cycle_enum";

--bun:split
DROP TYPE IF EXISTS "invoice_split_key_enum";

--bun:split
DROP TYPE IF EXISTS "invoice_section_key_enum";

--bun:split
DROP TYPE IF EXISTS "invoice_detail_enum";
