CREATE OR REPLACE FUNCTION pg_temp.tariff_rate_unit(template_name text)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE
    AS $$
    SELECT
        CASE template_name
        WHEN 'Per Mile' THEN
            '$/mi'
        WHEN 'Per Stop' THEN
            '$/stop'
        WHEN 'Per Pound' THEN
            '$/lb'
        WHEN 'Per Pallet' THEN
            '$/pallet'
        WHEN 'Per Piece' THEN
            '$/piece'
        WHEN 'Per Linear Foot' THEN
            '$/ft'
        WHEN 'Per CWT (Hundredweight)' THEN
            '$/cwt'
        WHEN 'Per Hour' THEN
            '$/h'
        WHEN 'Percent of Sell Rate' THEN
            '%'
        ELSE
            '$'
        END
$$;

CREATE OR REPLACE FUNCTION pg_temp.tariff_quantity(amount numeric, unit text)
    RETURNS jsonb
    LANGUAGE sql
    IMMUTABLE
    AS $$
    SELECT
        jsonb_build_object('$q', trim_scale(amount)::text, 'u', unit)
$$;

ALTER TABLE "rate_agreements"
    ADD COLUMN IF NOT EXISTS "inputs" jsonb NOT NULL DEFAULT '{}'::jsonb;

UPDATE
    "rate_agreements"
SET
    "inputs" = jsonb_build_object('minimumCharge', pg_temp.tariff_quantity("default_min_charge", '$'))
WHERE
    "default_min_charge" IS NOT NULL;

ALTER TABLE "rate_agreements"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreements_charges",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreements_precision",
    DROP COLUMN IF EXISTS "default_min_charge",
    DROP COLUMN IF EXISTS "default_max_charge",
    DROP COLUMN IF EXISTS "rounding_mode",
    DROP COLUMN IF EXISTS "rounding_precision",
    ADD CONSTRAINT "chk_rate_agreements_inputs" CHECK (jsonb_typeof("inputs") = 'object');

ALTER TABLE "rate_agreement_versions"
    ADD COLUMN IF NOT EXISTS "inputs" jsonb NOT NULL DEFAULT '{}'::jsonb;

UPDATE
    "rate_agreement_versions"
SET
    "inputs" = jsonb_build_object('minimumCharge', pg_temp.tariff_quantity("default_min_charge", '$'))
WHERE
    "default_min_charge" IS NOT NULL;

ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "default_min_charge",
    DROP COLUMN IF EXISTS "default_max_charge",
    DROP COLUMN IF EXISTS "rounding_mode",
    DROP COLUMN IF EXISTS "rounding_precision",
    ADD CONSTRAINT "chk_rate_agreement_versions_inputs" CHECK (jsonb_typeof("inputs") = 'object');

ALTER TABLE "rate_agreement_rules"
    ADD COLUMN IF NOT EXISTS "inputs" jsonb NOT NULL DEFAULT '{}'::jsonb;

UPDATE
    "rate_agreement_rules" ragr
SET
    "formula_template_id" = rmx."formula_template_id"
FROM
    "rate_matrices" rmx
WHERE
    ragr."formula_template_id" IS NULL
    AND rmx."id" = ragr."rate_matrix_id"
    AND rmx."organization_id" = ragr."organization_id"
    AND rmx."business_unit_id" = ragr."business_unit_id";

DELETE FROM "rate_agreement_rules"
WHERE "formula_template_id" IS NULL;

UPDATE
    "rate_agreement_rules" ragr
SET
    "inputs" = jsonb_strip_nulls(jsonb_build_object('rate', CASE WHEN ragr."rate" IS NOT NULL THEN
                    pg_temp.tariff_quantity(ragr."rate", pg_temp.tariff_rate_unit(ft."name"))
                END, 'minimumCharge', CASE WHEN ragr."min_charge" IS NOT NULL THEN
                    pg_temp.tariff_quantity(ragr."min_charge", '$')
                END))
FROM
    "formula_templates" ft
WHERE
    ft."id" = ragr."formula_template_id"
    AND ft."organization_id" = ragr."organization_id"
    AND ft."business_unit_id" = ragr."business_unit_id";

ALTER TABLE "rate_agreement_rules"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_pricing_method",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_fixed_class",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_charges",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_discount",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_rate",
    DROP CONSTRAINT IF EXISTS "fk_rate_agreement_rules_matrix",
    DROP CONSTRAINT IF EXISTS "fk_rate_agreement_rules_density_scale",
    DROP COLUMN IF EXISTS "rate",
    DROP COLUMN IF EXISTS "rate_matrix_id",
    DROP COLUMN IF EXISTS "freight_class_source",
    DROP COLUMN IF EXISTS "fixed_freight_class",
    DROP COLUMN IF EXISTS "density_scale_id",
    DROP COLUMN IF EXISTS "discount_percent",
    DROP COLUMN IF EXISTS "absolute_min_charge",
    DROP COLUMN IF EXISTS "allow_deficit_rating",
    DROP COLUMN IF EXISTS "min_charge",
    DROP COLUMN IF EXISTS "max_charge",
    DROP COLUMN IF EXISTS "min_billable_distance",
    DROP COLUMN IF EXISTS "rounding_mode",
    ALTER COLUMN "formula_template_id" SET NOT NULL,
    ADD CONSTRAINT "chk_rate_agreement_rules_inputs" CHECK (jsonb_typeof("inputs") = 'object');

DROP TABLE IF EXISTS "rate_agreement_rule_breaks";

DROP TYPE IF EXISTS "rate_freight_class_source_enum";

ALTER TABLE "rate_agreement_accessorials"
    ADD COLUMN IF NOT EXISTS "condition" text;

UPDATE
    "rate_agreement_accessorials"
SET
    "auto_apply" = FALSE
WHERE
    "apply_condition" IS NOT NULL;

ALTER TABLE "rate_agreement_accessorials"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_accessorials_condition",
    DROP CONSTRAINT IF EXISTS "fk_rate_agreement_accessorials_formula_template",
    DROP COLUMN IF EXISTS "apply_condition",
    DROP COLUMN IF EXISTS "condition_language_version",
    DROP COLUMN IF EXISTS "formula_template_id",
    ADD CONSTRAINT "chk_rate_agreement_accessorials_condition" CHECK ("condition" IS NULL OR "auto_apply");

ALTER TABLE "rate_matrices"
    ADD COLUMN IF NOT EXISTS "value_type" varchar(32);

UPDATE
    "rate_matrices" rmx
SET
    "value_type" = CASE pg_temp.tariff_rate_unit(ft."name")
    WHEN '$' THEN
        'Money'
    WHEN '%' THEN
        'Percent'
    ELSE
        'Money/' || substr(pg_temp.tariff_rate_unit(ft."name"), 3)
    END
FROM
    "formula_templates" ft
WHERE
    ft."id" = rmx."formula_template_id"
    AND ft."organization_id" = rmx."organization_id"
    AND ft."business_unit_id" = rmx."business_unit_id";

UPDATE
    "rate_matrices"
SET
    "value_type" = 'Money'
WHERE
    "value_type" IS NULL;

ALTER TABLE "rate_matrices"
    DROP CONSTRAINT IF EXISTS "fk_rate_matrices_formula_template",
    DROP CONSTRAINT IF EXISTS "chk_rate_matrices_precision",
    DROP COLUMN IF EXISTS "formula_template_id",
    DROP COLUMN IF EXISTS "currency",
    DROP COLUMN IF EXISTS "rounding_mode",
    DROP COLUMN IF EXISTS "rounding_precision",
    ALTER COLUMN "value_type" SET NOT NULL,
    ADD CONSTRAINT "chk_rate_matrices_value_type" CHECK ("value_type" IN ('Money', 'Percent', 'Number', 'Int') OR "value_type" ~ '^Money/(lb|cwt|kg|t|ton|mi|km|in|ft|m|ft3|m3|gal|l|min|h|d|pallet|piece|stop)$');

DROP TABLE IF EXISTS "rate_density_scale_tiers";

DROP TABLE IF EXISTS "rate_density_scales";

ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "tariff_inputs" jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "freight_lines" jsonb NOT NULL DEFAULT '[]'::jsonb;

UPDATE
    "shipments" sp
SET
    "tariff_inputs" = jsonb_build_object('rate', pg_temp.tariff_quantity(sp."base_rate", pg_temp.tariff_rate_unit(ft."name")))
FROM
    "formula_templates" ft
WHERE
    sp."base_rate" <> 0
    AND ft."id" = sp."formula_template_id"
    AND ft."organization_id" = sp."organization_id"
    AND ft."business_unit_id" = sp."business_unit_id";

UPDATE
    "shipments"
SET
    "freight_lines" = jsonb_build_array(jsonb_build_object('label', 'Freight charge', 'amount', trim_scale("freight_charge_amount")::text)),
    "rating_detail" = NULL
WHERE
    "freight_charge_amount" <> 0;

UPDATE
    "shipments"
SET
    "rating_detail" = NULL
WHERE
    "rating_detail" IS NOT NULL;

ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "base_rate",
    DROP COLUMN IF EXISTS "rating_unit",
    ADD CONSTRAINT "chk_shipments_tariff_inputs" CHECK (jsonb_typeof("tariff_inputs") = 'object'),
    ADD CONSTRAINT "chk_shipments_freight_lines" CHECK (jsonb_typeof("freight_lines") = 'array');

DELETE FROM "edi_source_context_fields"
WHERE "id" IN ('ediscf_x12_204_base_rate', 'ediscf_x12_204_rating_unit');

ALTER TABLE "additional_charges"
    ADD COLUMN IF NOT EXISTS "is_tariff" boolean NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "label" varchar(255),
    ADD COLUMN IF NOT EXISTS "quantity" numeric(19, 6),
    ADD COLUMN IF NOT EXISTS "rate" numeric(19, 6),
    ADD COLUMN IF NOT EXISTS "rate_unit" varchar(32);

ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_single_owner";

ALTER TABLE "additional_charges"
    ADD CONSTRAINT "chk_additional_charges_single_owner" CHECK (NOT "is_system_generated" OR (("fuel_surcharge_program_id" IS NOT NULL)::int + "is_detention"::int + "is_tariff"::int + ("rate_agreement_accessorial_id" IS NOT NULL)::int) <= 1),
    ADD CONSTRAINT "chk_additional_charges_tariff_system" CHECK (NOT "is_tariff" OR "is_system_generated");

UPDATE
    "invoice_lines"
SET
    "rate_unit" = CASE "rate_unit"
    WHEN 'Mile' THEN
        '$/mi'
    WHEN 'Hour' THEN
        '$/h'
    WHEN 'Day' THEN
        '$/d'
    WHEN 'Stop' THEN
        '$/stop'
    ELSE
        "rate_unit"
    END
WHERE
    "rate_unit" IS NOT NULL;

ALTER TABLE "invoice_lines"
    ALTER COLUMN "rate_unit" TYPE varchar(32);

ALTER TABLE "invoices"
    ADD COLUMN IF NOT EXISTS "rating_notes" text[];

SELECT
    trenova_rls.reconcile();
