CREATE TYPE "rate_matrix_value_kind_enum" AS ENUM(
    'FlatRate',
    'PerMile',
    'PerCwt',
    'PerPiece',
    'PerStop',
    'Percent',
    'Discount',
    'MinimumOnly'
);

--bun:split
CREATE TYPE "rate_agreement_basis_enum" AS ENUM(
    'Flat',
    'PerMile',
    'PerCwt',
    'PerPiece',
    'PerStop',
    'PerPallet',
    'PerLinearFoot',
    'PerHour',
    'Percent',
    'Matrix',
    'Formula'
);

--bun:split
CREATE TYPE "rate_agreement_percent_basis_enum" AS ENUM(
    'Linehaul',
    'LinehaulPlusAccessorials',
    'SellRate'
);

--bun:split
ALTER TABLE "rate_matrices"
    ADD COLUMN IF NOT EXISTS "value_kind" rate_matrix_value_kind_enum NOT NULL DEFAULT 'FlatRate';

--bun:split
-- Reverse the backfill by template name where the standard names still apply;
-- anything else stays at the flat default, which is the least surprising read
-- of an unknown template.
UPDATE "rate_matrices" rmx
SET "value_kind" = CASE ft."name"
        WHEN 'Per Mile' THEN 'PerMile'::rate_matrix_value_kind_enum
        WHEN 'Per CWT (Hundredweight)' THEN 'PerCwt'::rate_matrix_value_kind_enum
        WHEN 'Per Piece' THEN 'PerPiece'::rate_matrix_value_kind_enum
        WHEN 'Per Stop' THEN 'PerStop'::rate_matrix_value_kind_enum
        ELSE 'FlatRate'::rate_matrix_value_kind_enum
    END
FROM "formula_templates" ft
WHERE ft."id" = rmx."formula_template_id"
    AND ft."organization_id" = rmx."organization_id"
    AND ft."business_unit_id" = rmx."business_unit_id";

--bun:split
ALTER TABLE "rate_matrices"
    DROP CONSTRAINT IF EXISTS "fk_rate_matrices_formula_template";

--bun:split
ALTER TABLE "rate_matrices"
    DROP COLUMN IF EXISTS "formula_template_id";

--bun:split
ALTER TABLE "rate_agreement_rules"
    ADD COLUMN IF NOT EXISTS "rating_basis" rate_agreement_basis_enum NOT NULL DEFAULT 'Formula',
    ADD COLUMN IF NOT EXISTS "percent_basis" rate_agreement_percent_basis_enum;

--bun:split
UPDATE "rate_agreement_rules"
SET "rating_basis" = 'Matrix'
WHERE "rate_matrix_id" IS NOT NULL;

--bun:split
UPDATE "rate_agreement_rules" ragr
SET "rating_basis" = CASE ft."name"
        WHEN 'Flat Rate' THEN 'Flat'::rate_agreement_basis_enum
        WHEN 'Per Mile' THEN 'PerMile'::rate_agreement_basis_enum
        WHEN 'Per CWT (Hundredweight)' THEN 'PerCwt'::rate_agreement_basis_enum
        WHEN 'Per Piece' THEN 'PerPiece'::rate_agreement_basis_enum
        WHEN 'Per Stop' THEN 'PerStop'::rate_agreement_basis_enum
        WHEN 'Per Pallet' THEN 'PerPallet'::rate_agreement_basis_enum
        WHEN 'Per Linear Foot' THEN 'PerLinearFoot'::rate_agreement_basis_enum
        WHEN 'Per Hour' THEN 'PerHour'::rate_agreement_basis_enum
        WHEN 'Percent of Sell Rate' THEN 'Percent'::rate_agreement_basis_enum
        ELSE 'Formula'::rate_agreement_basis_enum
    END
FROM "formula_templates" ft
WHERE ft."id" = ragr."formula_template_id"
    AND ft."organization_id" = ragr."organization_id"
    AND ft."business_unit_id" = ragr."business_unit_id";

--bun:split
UPDATE "rate_agreement_rules"
SET "percent_basis" = 'SellRate'
WHERE "rating_basis" = 'Percent';

--bun:split
-- Rules restated on a canned basis no longer carry a template reference.
UPDATE "rate_agreement_rules"
SET "formula_template_id" = NULL
WHERE "rating_basis" NOT IN ('Formula');

--bun:split
ALTER TABLE "rate_agreement_rules"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_pricing_method";

--bun:split
ALTER TABLE "rate_agreement_rules"
    ADD CONSTRAINT "chk_rate_agreement_rules_matrix_basis" CHECK (("rating_basis" = 'Matrix') = ("rate_matrix_id" IS NOT NULL)) NOT VALID,
    ADD CONSTRAINT "chk_rate_agreement_rules_formula_basis" CHECK (("rating_basis" = 'Formula') = ("formula_template_id" IS NOT NULL)) NOT VALID,
    ADD CONSTRAINT "chk_rate_agreement_rules_percent_basis" CHECK ("rating_basis" <> 'Percent' OR "percent_basis" IS NOT NULL) NOT VALID,
    ADD CONSTRAINT "chk_rate_agreement_rules_sell_rate_basis" CHECK ("percent_basis" <> 'SellRate' OR "party_type" = 'Carrier') NOT VALID;
