DROP TABLE IF EXISTS "formula_template_version_imports";

--bun:split
DROP TABLE IF EXISTS "formula_library_reviews";

--bun:split
DROP TABLE IF EXISTS "formula_library_versions";

--bun:split
DROP TABLE IF EXISTS "formula_libraries";

--bun:split
CREATE TYPE "formula_template_type_enum" AS ENUM('FreightCharge', 'AccessorialCharge');

--bun:split
CREATE TYPE "rate_rounding_mode_enum" AS ENUM('HalfUp', 'HalfEven', 'Up', 'Down', 'None');

--bun:split
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
    "recorded_on_version" bigint,
    "recorded_by_id" varchar(100),
    "recorded_at" bigint,
    "recorded_content" jsonb,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_formula_template_test_cases" PRIMARY KEY ("id"),
    CONSTRAINT "uk_formula_template_test_cases_name" UNIQUE ("template_id", "organization_id", "business_unit_id", "name"),
    CONSTRAINT "fk_formula_template_test_cases_template" FOREIGN KEY ("template_id", "organization_id", "business_unit_id")
        REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON DELETE CASCADE
);

--bun:split
ALTER TABLE "formula_template_versions"
    DROP CONSTRAINT IF EXISTS "ck_formula_template_versions_test_count",
    DROP COLUMN IF EXISTS "source",
    DROP COLUMN IF EXISTS "inputs",
    DROP COLUMN IF EXISTS "imports",
    DROP COLUMN IF EXISTS "test_count",
    ADD COLUMN IF NOT EXISTS "type" formula_template_type_enum NOT NULL DEFAULT 'FreightCharge',
    ADD COLUMN IF NOT EXISTS "expression" text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "schema_id" varchar(100) NOT NULL DEFAULT 'shipment',
    ADD COLUMN IF NOT EXISTS "language_version" smallint NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS "variable_definitions" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "breakdown_definitions" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "max_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    ADD COLUMN IF NOT EXISTS "rounding_precision" smallint NOT NULL DEFAULT 2;

--bun:split
ALTER TABLE "formula_templates"
    DROP CONSTRAINT IF EXISTS "ck_formula_templates_test_count",
    DROP CONSTRAINT IF EXISTS "ck_formula_templates_inputs_array",
    DROP CONSTRAINT IF EXISTS "ck_formula_templates_imports_array",
    DROP COLUMN IF EXISTS "source",
    DROP COLUMN IF EXISTS "inputs",
    DROP COLUMN IF EXISTS "imports",
    DROP COLUMN IF EXISTS "test_count",
    ADD COLUMN IF NOT EXISTS "type" formula_template_type_enum NOT NULL DEFAULT 'FreightCharge',
    ADD COLUMN IF NOT EXISTS "expression" text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "schema_id" varchar(100) NOT NULL DEFAULT 'shipment',
    ADD COLUMN IF NOT EXISTS "language_version" smallint NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS "variable_definitions" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "breakdown_definitions" jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN IF NOT EXISTS "min_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "max_charge" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    ADD COLUMN IF NOT EXISTS "rounding_precision" smallint NOT NULL DEFAULT 2;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_templates_type_status" ON "formula_templates"("type", "status");

--bun:split
SELECT trenova_rls.reconcile();
