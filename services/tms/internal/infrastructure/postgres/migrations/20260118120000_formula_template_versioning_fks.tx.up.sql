--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
ALTER TABLE "formula_template_versions"
    ADD CONSTRAINT "fk_formula_template_versions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON DELETE RESTRICT;

--bun:split
ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS "source_template_id" varchar(100);

--bun:split
ALTER TABLE "formula_templates"
    ADD CONSTRAINT "fk_formula_templates_source_template" FOREIGN KEY ("source_template_id", "organization_id", "business_unit_id") REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE SET NULL;

