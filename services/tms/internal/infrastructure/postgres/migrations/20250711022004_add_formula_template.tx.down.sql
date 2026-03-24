--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "shipments"
    DROP CONSTRAINT IF EXISTS "fk_shipments_formula_template";

--bun:split
DROP INDEX IF EXISTS "idx_shipments_formula_template";

--bun:split
ALTER TABLE "shipments"
    DROP COLUMN IF EXISTS "formula_template_id";

--bun:split
DROP TABLE IF EXISTS "formula_templates";

--bun:split
DROP TYPE IF EXISTS "formula_template_type_enum";
DROP TYPE IF EXISTS "formula_template_status_enum";
