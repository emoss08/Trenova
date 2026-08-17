ALTER TABLE "billing_controls"
    DROP COLUMN IF EXISTS "enforce_margin_floor",
    DROP COLUMN IF EXISTS "require_rate_override_reason",
    DROP COLUMN IF EXISTS "fallback_formula_template_id",
    DROP COLUMN IF EXISTS "unrated_shipment_disposition";

--bun:split
DROP INDEX IF EXISTS "idx_additional_charges_agreement_accessorial";

--bun:split
ALTER TABLE "additional_charges"
    DROP CONSTRAINT IF EXISTS "chk_additional_charges_single_owner";

--bun:split
ALTER TABLE "additional_charges"
    DROP COLUMN IF EXISTS "rate_quote_id",
    DROP COLUMN IF EXISTS "rate_agreement_accessorial_id";

--bun:split
-- Restoring the requirement means any shipment rated purely from an agreement
-- would violate it, so the rollback first backfills those rows from the
-- template the quote recorded, where there was one.
UPDATE
    "shipments" s
SET
    "formula_template_id" = q."formula_template_id"
FROM
    "rate_quotes" q
WHERE
    s."rate_quote_id" = q."id"
    AND s."organization_id" = q."organization_id"
    AND s."business_unit_id" = q."business_unit_id"
    AND s."formula_template_id" IS NULL
    AND q."formula_template_id" IS NOT NULL;

--bun:split
-- Any shipment still without a template was priced purely by an agreement and
-- has no template to fall back to. Rather than invent one or delete the
-- shipment, this fails the rollback loudly: the whole migration runs in a
-- transaction, so nothing is lost and an operator can decide what those
-- shipments should be rated with.
ALTER TABLE "shipments"
    ALTER COLUMN "formula_template_id" SET NOT NULL;

--bun:split
DROP INDEX IF EXISTS "idx_shipments_rate_agreement";

--bun:split
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "rate_locked",
    DROP COLUMN IF EXISTS "rate_override_at",
    DROP COLUMN IF EXISTS "rate_override_by_id",
    DROP COLUMN IF EXISTS "rate_override_reason",
    DROP COLUMN IF EXISTS "rate_override_amount",
    DROP COLUMN IF EXISTS "rate_agreement_rule_id",
    DROP COLUMN IF EXISTS "rate_agreement_id",
    DROP COLUMN IF EXISTS "rate_quote_id";

--bun:split
DROP TABLE IF EXISTS "rate_quotes";

--bun:split
DROP TABLE IF EXISTS "rate_agreement_fuel_bindings";

--bun:split
DROP TABLE IF EXISTS "rate_agreement_accessorials";

--bun:split
DROP TABLE IF EXISTS "rate_agreement_rule_breaks";

--bun:split
DROP TABLE IF EXISTS "rate_agreement_rules";

--bun:split
DROP TABLE IF EXISTS "rate_agreement_versions";

--bun:split
DROP TABLE IF EXISTS "rate_agreements";

--bun:split
DROP TYPE IF EXISTS "unrated_shipment_disposition_enum";

--bun:split
DROP TYPE IF EXISTS "rate_quote_status_enum";

--bun:split
DROP TYPE IF EXISTS "rate_quote_outcome_enum";

--bun:split
DROP TYPE IF EXISTS "rate_quote_purpose_enum";

--bun:split
DROP TYPE IF EXISTS "rate_freight_class_source_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_direction_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_percent_basis_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_basis_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_rule_status_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_status_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_type_enum";

--bun:split
DROP TYPE IF EXISTS "rate_agreement_party_type_enum";
