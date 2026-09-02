--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP INDEX IF EXISTS "idx_accessorial_charges_formula_template";

--bun:split
DROP INDEX IF EXISTS "idx_formula_templates_tenant_created";

--bun:split
DROP INDEX IF EXISTS "uk_formula_templates_name";
