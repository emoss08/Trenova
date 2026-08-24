-- Every rate in Trenova is now priced by a formula template. The canned
-- rating-method enums are retired: a rate matrix names the template that says
-- what its cells mean, and an agreement rule prices through exactly one of a
-- formula template or a rate matrix.
ALTER TABLE "rate_matrices"
    ADD COLUMN IF NOT EXISTS "formula_template_id" varchar(100);

--bun:split
-- Best-effort backfill: map each retired value kind onto the organization's
-- standard template of the same meaning. A matrix whose organization lacks the
-- named template stays NULL and must be repointed before it can rate anything;
-- the engine refuses it cleanly rather than guessing.
UPDATE "rate_matrices" rmx
SET "formula_template_id" = (
    SELECT ft."id"
    FROM "formula_templates" ft
    WHERE ft."organization_id" = rmx."organization_id"
        AND ft."business_unit_id" = rmx."business_unit_id"
        AND ft."status" = 'Active'
        AND ft."name" = CASE rmx."value_kind"
            WHEN 'PerMile' THEN 'Per Mile'
            WHEN 'PerCwt' THEN 'Per CWT (Hundredweight)'
            WHEN 'PerPiece' THEN 'Per Piece'
            WHEN 'PerStop' THEN 'Per Stop'
            ELSE 'Flat Rate'
        END
    ORDER BY ft."created_at" ASC
    LIMIT 1
)
WHERE rmx."formula_template_id" IS NULL;

--bun:split
ALTER TABLE "rate_matrices"
    ADD CONSTRAINT "fk_rate_matrices_formula_template" FOREIGN KEY ("formula_template_id", "organization_id", "business_unit_id") REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT;

--bun:split
-- Rules on a canned basis backfill the same way. Matrix and Formula rules
-- already carry their pricing reference and are left alone.
UPDATE "rate_agreement_rules" ragr
SET "formula_template_id" = (
    SELECT ft."id"
    FROM "formula_templates" ft
    WHERE ft."organization_id" = ragr."organization_id"
        AND ft."business_unit_id" = ragr."business_unit_id"
        AND ft."status" = 'Active'
        AND ft."name" = CASE ragr."rating_basis"
            WHEN 'PerMile' THEN 'Per Mile'
            WHEN 'PerCwt' THEN 'Per CWT (Hundredweight)'
            WHEN 'PerPiece' THEN 'Per Piece'
            WHEN 'PerStop' THEN 'Per Stop'
            WHEN 'PerPallet' THEN 'Per Pallet'
            WHEN 'PerLinearFoot' THEN 'Per Linear Foot'
            WHEN 'PerHour' THEN 'Per Hour'
            WHEN 'Percent' THEN 'Percent of Sell Rate'
            ELSE 'Flat Rate'
        END
    ORDER BY ft."created_at" ASC
    LIMIT 1
)
WHERE ragr."formula_template_id" IS NULL
    AND ragr."rating_basis" NOT IN ('Matrix', 'Formula');

--bun:split
ALTER TABLE "rate_agreement_rules"
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_matrix_basis",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_formula_basis",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_percent_basis",
    DROP CONSTRAINT IF EXISTS "chk_rate_agreement_rules_sell_rate_basis";

--bun:split
-- Exactly one pricing method. NOT VALID because a rule the backfill could not
-- map has neither, and refusing the migration over it would trade a rule that
-- cannot price for a schema that cannot move; new writes are held to it.
ALTER TABLE "rate_agreement_rules"
    ADD CONSTRAINT "chk_rate_agreement_rules_pricing_method" CHECK (("formula_template_id" IS NOT NULL) <> ("rate_matrix_id" IS NOT NULL)) NOT VALID;

--bun:split
ALTER TABLE "rate_agreement_rules"
    DROP COLUMN IF EXISTS "rating_basis",
    DROP COLUMN IF EXISTS "percent_basis";

--bun:split
ALTER TABLE "rate_matrices"
    DROP COLUMN IF EXISTS "value_kind";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_basis_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_percent_basis_enum";

--bun:split
DROP TYPE IF EXISTS "rate_matrix_value_kind_enum";
