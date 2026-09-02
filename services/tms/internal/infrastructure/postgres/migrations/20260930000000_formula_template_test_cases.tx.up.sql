--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "formula_template_test_cases"(
    "id" varchar(100) NOT NULL,
    "template_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "variables" jsonb NOT NULL DEFAULT '{}',
    "expected_amount" numeric(19, 4) NOT NULL,
    "tolerance" numeric(19, 4) NOT NULL DEFAULT 0.01,
    "version" bigint NOT NULL DEFAULT 0,
    "created_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_formula_template_test_cases" PRIMARY KEY ("id"),
    CONSTRAINT "uk_formula_template_test_cases_name" UNIQUE ("template_id", "organization_id", "business_unit_id", "name"),
    CONSTRAINT "fk_formula_template_test_cases_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_test_cases_organization" FOREIGN KEY ("organization_id")
        REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_formula_template_test_cases_business_unit" FOREIGN KEY ("business_unit_id")
        REFERENCES "business_units"("id") ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_template_test_cases_template" ON "formula_template_test_cases"("template_id", "organization_id", "business_unit_id");
