ALTER TABLE "invoices"
    DROP COLUMN IF EXISTS "rating_notes";

--bun:split
UPDATE
    "invoice_lines"
SET
    "rate_unit" = CASE "rate_unit"
    WHEN '$/mi' THEN
        'Mile'
    WHEN '$/h' THEN
        'Hour'
    WHEN '$/d' THEN
        'Day'
    WHEN '$/stop' THEN
        'Stop'
    ELSE
        NULL
    END
WHERE
    "rate_unit" IS NOT NULL;

--bun:split
ALTER TABLE "invoice_lines"
    ALTER COLUMN "rate_unit" TYPE varchar(20);

--bun:split
DELETE FROM "additional_charges"
WHERE "is_tariff";

--bun:split
ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_tariff_system",
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_single_owner";

--bun:split
ALTER TABLE "additional_charges"
    DROP COLUMN IF EXISTS "is_tariff",
    DROP COLUMN IF EXISTS "label",
    DROP COLUMN IF EXISTS "quantity",
    DROP COLUMN IF EXISTS "rate",
    DROP COLUMN IF EXISTS "rate_unit",
    ADD CONSTRAINT "chk_additional_charges_single_owner" CHECK (NOT "is_system_generated" OR (("fuel_surcharge_program_id" IS NOT NULL)::int + "is_detention"::int + ("rate_agreement_accessorial_id" IS NOT NULL)::int) <= 1);

--bun:split
INSERT INTO "edi_source_context_fields"("id", "schema_id", "path", "source_kind", "data_type", "repeated", "repeat_path", "parent_path", "display_name", "description", "status")
SELECT
    v."id",
    s."id",
    v."path",
    'shipment',
    v."data_type",
    FALSE,
    NULL,
    'shipment',
    v."display_name",
    v."description",
    'Active'
FROM (
    VALUES ('ediscf_x12_204_base_rate', 'shipment.baseRate', 'decimal', 'Base Rate', 'Base rate amount.'),
        ('ediscf_x12_204_rating_unit', 'shipment.ratingUnit', 'integer', 'Rating Unit', 'Rating unit.')) AS v("id", "path", "data_type", "display_name", "description")
    CROSS JOIN LATERAL (
        SELECT
            f."schema_id" AS "id"
        FROM
            "edi_source_context_fields" f
        WHERE
            f."id" = 'ediscf_x12_204_shipment_id') s
ON CONFLICT DO NOTHING;

--bun:split
ALTER TABLE "shipments"
    DROP CONSTRAINT IF EXISTS "chk_shipments_tariff_inputs",
    DROP CONSTRAINT IF EXISTS "chk_shipments_freight_lines",
    ADD COLUMN IF NOT EXISTS "base_rate" numeric(19, 4) NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS "rating_unit" integer NOT NULL DEFAULT 1 CHECK ("rating_unit" > 0);

--bun:split
UPDATE
    "shipments"
SET
    "base_rate" = ("tariff_inputs" -> 'rate' ->> '$q')::numeric
WHERE
    "tariff_inputs" ? 'rate'
    AND jsonb_typeof("tariff_inputs" -> 'rate') = 'object';

--bun:split
UPDATE
    "shipments"
SET
    "rating_detail" = NULL
WHERE
    "rating_detail" IS NOT NULL;

--bun:split
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "tariff_inputs",
    DROP COLUMN IF EXISTS "freight_lines";

--bun:split
CREATE TABLE IF NOT EXISTS "rate_density_scales"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(64) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "effective_from" bigint NOT NULL,
    "is_org_default" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_density_scales" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_density_scales_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scales_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scales_code" ON "rate_density_scales"("organization_id", "business_unit_id", lower("code"));

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scales_default" ON "rate_density_scales"("organization_id", "business_unit_id")
WHERE
    "is_org_default";

--bun:split
CREATE TABLE IF NOT EXISTS "rate_density_scale_tiers"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_density_scale_id" varchar(100) NOT NULL,
    "from_pcf" numeric(10, 4) NOT NULL,
    "to_pcf" numeric(10, 4),
    "freight_class" freight_class_enum NOT NULL,
    "sort_order" smallint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_density_scale_tiers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_density_scale_tiers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scale_tiers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scale_tiers_scale" FOREIGN KEY ("rate_density_scale_id", "organization_id", "business_unit_id") REFERENCES "rate_density_scales"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_density_scale_tiers_from" CHECK ("from_pcf" >= 0),
    CONSTRAINT "chk_rate_density_scale_tiers_band" CHECK ("to_pcf" IS NULL OR "to_pcf" > "from_pcf")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scale_tiers_from" ON "rate_density_scale_tiers"("organization_id", "business_unit_id", "rate_density_scale_id", "from_pcf");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_density_scale_tiers_scale" ON "rate_density_scale_tiers"("organization_id", "business_unit_id", "rate_density_scale_id", "from_pcf");

--bun:split
ALTER TABLE "rate_matrices"
    DROP CONSTRAINT IF EXISTS "chk_rate_matrices_value_type",
    ADD COLUMN IF NOT EXISTS "formula_template_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "currency" varchar(3) NOT NULL DEFAULT 'USD',
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    ADD COLUMN IF NOT EXISTS "rounding_precision" smallint NOT NULL DEFAULT 2,
    ADD CONSTRAINT "chk_rate_matrices_precision" CHECK ("rounding_precision" BETWEEN 0 AND 6),
    ADD CONSTRAINT "fk_rate_matrices_formula_template" FOREIGN KEY ("formula_template_id", "organization_id", "business_unit_id") REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    DROP COLUMN IF EXISTS "value_type";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_matrices_formula_template" ON "rate_matrices"("formula_template_id", "organization_id", "business_unit_id")
WHERE
    "formula_template_id" IS NOT NULL;

--bun:split
ALTER TABLE "rate_agreement_accessorials"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_accessorials_condition",
    ADD COLUMN IF NOT EXISTS "apply_condition" text,
    ADD COLUMN IF NOT EXISTS "condition_language_version" smallint NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS "formula_template_id" varchar(100),
    ADD CONSTRAINT "fk_rate_agreement_accessorials_formula_template" FOREIGN KEY ("formula_template_id", "organization_id", "business_unit_id") REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "chk_rate_agreement_accessorials_condition" CHECK ("apply_condition" IS NULL OR "auto_apply"),
    DROP COLUMN IF EXISTS "condition";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_accessorials_formula_template" ON "rate_agreement_accessorials"("formula_template_id", "organization_id", "business_unit_id")
WHERE
    "formula_template_id" IS NOT NULL;

--bun:split
CREATE TYPE "rate_freight_class_source_enum" AS ENUM(
    'Commodity',
    'Density',
    'Fixed'
);

--bun:split
ALTER TABLE "rate_agreement_rules"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_inputs",
    ALTER COLUMN "formula_template_id" DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS "rate" numeric(19, 6),
    ADD COLUMN IF NOT EXISTS "rate_matrix_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "freight_class_source" rate_freight_class_source_enum NOT NULL DEFAULT 'Commodity',
    ADD COLUMN IF NOT EXISTS "fixed_freight_class" freight_class_enum,
    ADD COLUMN IF NOT EXISTS "density_scale_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "discount_percent" numeric(9, 4),
    ADD COLUMN IF NOT EXISTS "absolute_min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "allow_deficit_rating" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "max_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "min_billable_distance" numeric(12, 2),
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum;

--bun:split
UPDATE
    "rate_agreement_rules"
SET
    "rate" = ("inputs" -> 'rate' ->> '$q')::numeric
WHERE
    jsonb_typeof("inputs" -> 'rate') = 'object';

--bun:split
UPDATE
    "rate_agreement_rules"
SET
    "min_charge" = ("inputs" -> 'minimumCharge' ->> '$q')::numeric
WHERE
    jsonb_typeof("inputs" -> 'minimumCharge') = 'object';

--bun:split
ALTER TABLE "rate_agreement_rules"
    DROP COLUMN IF EXISTS "inputs",
    ADD CONSTRAINT "fk_rate_agreement_rules_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id") REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "fk_rate_agreement_rules_density_scale" FOREIGN KEY ("density_scale_id", "organization_id", "business_unit_id") REFERENCES "rate_density_scales"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    ADD CONSTRAINT "chk_rate_agreement_rules_pricing_method" CHECK (("formula_template_id" IS NOT NULL) <> ("rate_matrix_id" IS NOT NULL)) NOT VALID,
    ADD CONSTRAINT "chk_rate_agreement_rules_fixed_class" CHECK (("freight_class_source" = 'Fixed') = ("fixed_freight_class" IS NOT NULL)),
    ADD CONSTRAINT "chk_rate_agreement_rules_charges" CHECK ("min_charge" IS NULL OR "max_charge" IS NULL OR "max_charge" >= "min_charge"),
    ADD CONSTRAINT "chk_rate_agreement_rules_discount" CHECK ("discount_percent" IS NULL OR ("discount_percent" >= 0 AND "discount_percent" <= 100)),
    ADD CONSTRAINT "chk_rate_agreement_rules_rate" CHECK ("rate" IS NULL OR "rate" >= 0);

--bun:split
CREATE TABLE IF NOT EXISTS "rate_agreement_rule_breaks"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_agreement_rule_id" varchar(100) NOT NULL,
    "from_weight" numeric(12, 2) NOT NULL,
    "to_weight" numeric(12, 2),
    "rate" numeric(19, 6) NOT NULL,
    "min_charge" numeric(19, 4),
    "label" varchar(50),
    "sort_order" smallint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_agreement_rule_breaks" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_rule_breaks_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rule_breaks_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rule_breaks_rule" FOREIGN KEY ("rate_agreement_rule_id", "organization_id", "business_unit_id") REFERENCES "rate_agreement_rules"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_agreement_rule_breaks_from" CHECK ("from_weight" >= 0),
    CONSTRAINT "chk_rate_agreement_rule_breaks_band" CHECK ("to_weight" IS NULL OR "to_weight" > "from_weight"),
    CONSTRAINT "chk_rate_agreement_rule_breaks_rate" CHECK ("rate" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_rule_breaks_from" ON "rate_agreement_rule_breaks"("organization_id", "business_unit_id", "rate_agreement_rule_id", "from_weight");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rule_breaks_rule" ON "rate_agreement_rule_breaks"("organization_id", "business_unit_id", "rate_agreement_rule_id", "from_weight");

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_versions_inputs",
    ADD COLUMN IF NOT EXISTS "default_min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "default_max_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    ADD COLUMN IF NOT EXISTS "rounding_precision" smallint NOT NULL DEFAULT 2;

--bun:split
UPDATE
    "rate_agreement_versions"
SET
    "default_min_charge" = ("inputs" -> 'minimumCharge' ->> '$q')::numeric
WHERE
    jsonb_typeof("inputs" -> 'minimumCharge') = 'object';

--bun:split
ALTER TABLE "rate_agreement_versions"
    DROP COLUMN IF EXISTS "inputs";

--bun:split
ALTER TABLE "rate_agreements"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreements_inputs",
    ADD COLUMN IF NOT EXISTS "default_min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "default_max_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    ADD COLUMN IF NOT EXISTS "rounding_precision" smallint NOT NULL DEFAULT 2,
    ADD CONSTRAINT "chk_rate_agreements_charges" CHECK ("default_min_charge" IS NULL OR "default_max_charge" IS NULL OR "default_max_charge" >= "default_min_charge"),
    ADD CONSTRAINT "chk_rate_agreements_precision" CHECK ("rounding_precision" BETWEEN 0 AND 6);

--bun:split
UPDATE
    "rate_agreements"
SET
    "default_min_charge" = ("inputs" -> 'minimumCharge' ->> '$q')::numeric
WHERE
    jsonb_typeof("inputs" -> 'minimumCharge') = 'object';

--bun:split
ALTER TABLE "rate_agreements"
    DROP COLUMN IF EXISTS "inputs";

--bun:split
SELECT
    trenova_rls.reconcile();
