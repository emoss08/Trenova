--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE UNIQUE INDEX IF NOT EXISTS "uk_formula_templates_name"
    ON "formula_templates"("organization_id", "business_unit_id", "name");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_templates_tenant_created"
    ON "formula_templates"("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_accessorial_charges_formula_template"
    ON "accessorial_charges"("formula_template_id", "organization_id", "business_unit_id")
    WHERE "formula_template_id" IS NOT NULL;
