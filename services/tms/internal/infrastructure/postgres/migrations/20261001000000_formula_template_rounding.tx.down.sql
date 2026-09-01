--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "formula_template_versions"
    DROP CONSTRAINT IF EXISTS "chk_formula_template_versions_rounding_precision";

--bun:split
ALTER TABLE "formula_template_versions"
    DROP COLUMN IF EXISTS "rounding_mode",
    DROP COLUMN IF EXISTS "rounding_precision";

--bun:split
ALTER TABLE "formula_templates"
    DROP CONSTRAINT IF EXISTS "chk_formula_templates_rounding_precision";

--bun:split
ALTER TABLE "formula_templates"
    DROP COLUMN IF EXISTS "rounding_mode",
    DROP COLUMN IF EXISTS "rounding_precision";
