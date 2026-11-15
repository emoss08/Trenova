-- A standard template is one of Trenova's catalogue rating methods (Flat Rate, Per Mile and the
-- rest of formulatemplateservice/standardcatalog), installed in every organization whatever its
-- plan and never changed there: the key names the catalogue entry it was installed from, and a
-- template without one is the organization's own. Every plan prices shipments with the standard
-- templates; the organization's own need the Rating add-on.
ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS "standard_key" varchar(64);

ALTER TABLE "formula_templates"
    ADD CONSTRAINT "ck_formula_templates_standard_key" CHECK ("standard_key" IS NULL OR "standard_key" <> '');

-- The templates the install action and the development seed already wrote from the catalogue are
-- marked by their catalogue name. One edited since (a later version, archived, back in review) or
-- copied from another template is the organization's own and stays unmarked; the next install
-- adds the catalogue entry beside it under another name.
UPDATE
    "formula_templates"
SET
    "standard_key" = CASE "name"
        WHEN 'Flat Rate' THEN 'flat_rate'
        WHEN 'Per Mile' THEN 'per_mile'
        WHEN 'Per Mile or Flat' THEN 'per_mile_or_flat'
        WHEN 'Per Stop' THEN 'per_stop'
        WHEN 'Per Pound' THEN 'per_pound'
        WHEN 'Per Pallet' THEN 'per_pallet'
        WHEN 'Per Piece' THEN 'per_piece'
        WHEN 'Per Linear Foot' THEN 'per_linear_foot'
        WHEN 'Per CWT (Hundredweight)' THEN 'per_cwt'
        WHEN 'Per Hour' THEN 'per_hour'
        WHEN 'Percent of Sell Rate' THEN 'percent_of_sell_rate'
        WHEN 'Weight Breaks (LTL)' THEN 'weight_breaks_ltl'
    END
WHERE
    "name" IN ('Flat Rate', 'Per Mile', 'Per Mile or Flat', 'Per Stop', 'Per Pound', 'Per Pallet', 'Per Piece', 'Per Linear Foot', 'Per CWT (Hundredweight)', 'Per Hour', 'Percent of Sell Rate', 'Weight Breaks (LTL)')
    AND "source_template_id" IS NULL
    AND "current_version_number" = 1
    AND "status" = 'Active';

COMMENT ON COLUMN "formula_templates"."standard_key" IS 'The catalogue entry a standard template was installed from; standard templates price shipments on every plan and are never changed. Null for a template the organization wrote';
