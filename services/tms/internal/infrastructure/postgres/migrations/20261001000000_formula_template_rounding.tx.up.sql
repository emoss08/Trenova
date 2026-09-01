--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    ADD COLUMN IF NOT EXISTS "rounding_precision" smallint NOT NULL DEFAULT 2;

--bun:split
ALTER TABLE "formula_templates"
    ADD CONSTRAINT "chk_formula_templates_rounding_precision"
    CHECK ("rounding_precision" BETWEEN 0 AND 4);

--bun:split
ALTER TABLE "formula_template_versions"
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    ADD COLUMN IF NOT EXISTS "rounding_precision" smallint NOT NULL DEFAULT 2;

--bun:split
ALTER TABLE "formula_template_versions"
    ADD CONSTRAINT "chk_formula_template_versions_rounding_precision"
    CHECK ("rounding_precision" BETWEEN 0 AND 4);
