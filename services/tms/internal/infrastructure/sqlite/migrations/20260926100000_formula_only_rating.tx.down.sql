-- Hand-completed SQLite translation of the formula-only rating rollback; the
-- converter cannot carry the PostgreSQL version's enum types or FROM-joined
-- updates, so this file no longer regenerates. See docs/databases.md.
-- Source: 20260926100000_formula_only_rating.tx.down.sql
--
-- Restores the retired columns so the code at the prior revision can run
-- again, reversing the template backfill by name the same best-effort way the
-- PostgreSQL rollback does. The check constraints the PostgreSQL version
-- re-adds cannot be expressed here: SQLite has no ALTER TABLE ADD CONSTRAINT.
ALTER TABLE "rate_matrices" ADD COLUMN "value_kind" TEXT NOT NULL DEFAULT 'FlatRate';

--bun:split
UPDATE "rate_matrices"
SET "value_kind" = COALESCE((
    SELECT CASE ft."name"
            WHEN 'Per Mile' THEN 'PerMile'
            WHEN 'Per CWT (Hundredweight)' THEN 'PerCwt'
            WHEN 'Per Piece' THEN 'PerPiece'
            WHEN 'Per Stop' THEN 'PerStop'
            ELSE 'FlatRate'
        END
    FROM "formula_templates" ft
    WHERE ft."id" = "rate_matrices"."formula_template_id"
      AND ft."organization_id" = "rate_matrices"."organization_id"
      AND ft."business_unit_id" = "rate_matrices"."business_unit_id"
), 'FlatRate');

--bun:split
ALTER TABLE "rate_matrices" DROP COLUMN "formula_template_id";

--bun:split
ALTER TABLE "rate_agreement_rules" ADD COLUMN "rating_basis" TEXT NOT NULL DEFAULT 'Formula';

--bun:split
ALTER TABLE "rate_agreement_rules" ADD COLUMN "percent_basis" TEXT;

--bun:split
UPDATE "rate_agreement_rules"
SET "rating_basis" = 'Matrix'
WHERE "rate_matrix_id" IS NOT NULL;

--bun:split
UPDATE "rate_agreement_rules"
SET "rating_basis" = (
    SELECT CASE ft."name"
            WHEN 'Flat Rate' THEN 'Flat'
            WHEN 'Per Mile' THEN 'PerMile'
            WHEN 'Per CWT (Hundredweight)' THEN 'PerCwt'
            WHEN 'Per Piece' THEN 'PerPiece'
            WHEN 'Per Stop' THEN 'PerStop'
            WHEN 'Per Pallet' THEN 'PerPallet'
            WHEN 'Per Linear Foot' THEN 'PerLinearFoot'
            WHEN 'Per Hour' THEN 'PerHour'
            WHEN 'Percent of Sell Rate' THEN 'Percent'
            ELSE 'Formula'
        END
    FROM "formula_templates" ft
    WHERE ft."id" = "rate_agreement_rules"."formula_template_id"
      AND ft."organization_id" = "rate_agreement_rules"."organization_id"
      AND ft."business_unit_id" = "rate_agreement_rules"."business_unit_id"
)
WHERE "formula_template_id" IS NOT NULL
  AND "rate_matrix_id" IS NULL
  AND EXISTS (
    SELECT 1
    FROM "formula_templates" ft
    WHERE ft."id" = "rate_agreement_rules"."formula_template_id"
      AND ft."organization_id" = "rate_agreement_rules"."organization_id"
      AND ft."business_unit_id" = "rate_agreement_rules"."business_unit_id"
);

--bun:split
UPDATE "rate_agreement_rules"
SET "percent_basis" = 'SellRate'
WHERE "rating_basis" = 'Percent';

--bun:split
UPDATE "rate_agreement_rules"
SET "formula_template_id" = NULL
WHERE "rating_basis" NOT IN ('Formula');
