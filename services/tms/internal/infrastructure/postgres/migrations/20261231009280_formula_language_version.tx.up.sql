-- The formula language each expression is written in, so a later language can
-- change what an expression means without changing what an existing template or
-- accessorial condition bills. Every stored expression is in version 1.
ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS "language_version" SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE "formula_template_versions"
    ADD COLUMN IF NOT EXISTS "language_version" SMALLINT NOT NULL DEFAULT 1;

ALTER TABLE "rate_agreement_accessorials"
    ADD COLUMN IF NOT EXISTS "condition_language_version" SMALLINT NOT NULL DEFAULT 1;
