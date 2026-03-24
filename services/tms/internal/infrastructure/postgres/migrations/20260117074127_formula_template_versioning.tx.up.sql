--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS "source_template_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "source_version_number" bigint,
    ADD COLUMN IF NOT EXISTS "current_version_number" bigint NOT NULL DEFAULT 1;

--bun:split
CREATE TABLE IF NOT EXISTS "formula_template_versions"(
    "id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "version_number" bigint NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "type" formula_template_type_enum NOT NULL,
    "expression" text NOT NULL,
    "status" formula_template_status_enum NOT NULL,
    "schema_id" varchar(100) NOT NULL,
    "variable_definitions" jsonb NOT NULL DEFAULT '[]',
    "metadata" jsonb,
    "change_message" text,
    "change_summary" jsonb,
    "created_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_formula_template_versions" PRIMARY KEY ("id"),
    CONSTRAINT "uk_formula_template_versions_template_version" UNIQUE ("template_id", "organization_id", "business_unit_id", "version_number"),
    CONSTRAINT "fk_formula_template_versions_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_versions_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_versions_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_template_versions_template" ON "formula_template_versions"("template_id", "organization_id", "business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_formula_template_versions_created_at" ON "formula_template_versions"("created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_templates_source" ON "formula_templates"("source_template_id")
WHERE "source_template_id" IS NOT NULL;
