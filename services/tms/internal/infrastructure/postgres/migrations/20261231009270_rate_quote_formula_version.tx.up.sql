-- A rate quote names the formula template version that priced it beside the
-- agreement version, so a template approved later cannot make a stored rating
-- unreadable.
ALTER TABLE "rate_quotes"
    ADD COLUMN IF NOT EXISTS "formula_template_version_number" BIGINT;
