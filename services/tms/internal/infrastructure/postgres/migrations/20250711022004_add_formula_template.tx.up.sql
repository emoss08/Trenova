--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TYPE "formula_template_status_enum" AS ENUM(
    'Active',
    'Inactive',
    'Draft'
);

CREATE TYPE "formula_template_type_enum" AS ENUM(
    'FreightCharge',
    'AccessorialCharge'
);

--bun:split
CREATE TABLE IF NOT EXISTS "formula_templates"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "type" formula_template_type_enum NOT NULL DEFAULT 'FreightCharge',
    "expression" text NOT NULL,
    "status" formula_template_status_enum NOT NULL DEFAULT 'Draft',
    "schema_id" varchar(100) NOT NULL DEFAULT 'shipment',
    "variable_definitions" jsonb NOT NULL DEFAULT '[]',
    "metadata" jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    CONSTRAINT "pk_formula_templates" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_formula_templates_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_formula_templates_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_formula_templates_type_status" ON "formula_templates"("type", "status");

CREATE INDEX IF NOT EXISTS "idx_formula_templates_bu_org" ON "formula_templates"("business_unit_id", "organization_id");

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "formula_template_id" varchar(100);

--bun:split
ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_formula_template" FOREIGN KEY ("formula_template_id", "business_unit_id", "organization_id") REFERENCES "formula_templates"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_formula_template" ON "shipments"("formula_template_id", "business_unit_id", "organization_id")
WHERE
    "formula_template_id" IS NOT NULL;

ALTER TABLE "formula_templates"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_formula_templates_search_vector ON "formula_templates" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION formula_templates_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'B') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS formula_templates_search_update ON "formula_templates";

CREATE TRIGGER formula_templates_search_update
    BEFORE INSERT OR UPDATE ON "formula_templates"
    FOR EACH ROW
    EXECUTE FUNCTION formula_templates_search_trigger();

--bun:split
UPDATE
    "formula_templates"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(name, '')), 'B') || setweight(to_tsvector('english', COALESCE(description, '')), 'B');

