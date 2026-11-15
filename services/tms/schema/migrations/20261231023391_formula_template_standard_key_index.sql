--keel:notx
-- One row per catalogue entry per organization: the install that runs when a picker or the
-- template list first opens relies on it when two servers install at once, and the picker of a plan
-- without the Rating add-on lists only these rows through it.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS "uk_formula_templates_standard_key" ON "formula_templates"("organization_id", "business_unit_id", "standard_key")
WHERE
    "standard_key" IS NOT NULL;
