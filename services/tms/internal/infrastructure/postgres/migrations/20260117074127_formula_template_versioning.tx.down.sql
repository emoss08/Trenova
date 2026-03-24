--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_formula_templates_source";

--bun:split
DROP TABLE IF EXISTS "formula_template_versions";

--bun:split
ALTER TABLE "formula_templates"
    DROP COLUMN IF EXISTS "source_template_id",
    DROP COLUMN IF EXISTS "source_version_number",
    DROP COLUMN IF EXISTS "current_version_number";
