ALTER TABLE "rate_agreement_accessorials"
    DROP COLUMN IF EXISTS "condition_language_version";

ALTER TABLE "formula_template_versions"
    DROP COLUMN IF EXISTS "language_version";

ALTER TABLE "formula_templates"
    DROP COLUMN IF EXISTS "language_version";
