CREATE TABLE IF NOT EXISTS "commodities"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "hazardous_material_id" varchar(100),
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" varchar(100) NOT NULL,
    "description" text NOT NULL,
    "min_temperature" integer,
    "max_temperature" integer,
    "max_quantity_per_shipment" float,
    "weight_per_unit" float,
    "linear_feet_per_unit" float,
    "freight_class" varchar(100),
    "dot_classification" varchar(100),
    "loading_instructions" text,
    "stackable" boolean NOT NULL DEFAULT FALSE,
    "fragile" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_commodities" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_commodities_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_commodities_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_commodities_hazardous_material" FOREIGN KEY ("hazardous_material_id", "business_unit_id", "organization_id") REFERENCES "hazardous_materials"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX "idx_commodities_name" ON "commodities"(lower("name"), "organization_id");

CREATE INDEX "idx_commodities_business_unit" ON "commodities"("business_unit_id");

CREATE INDEX "idx_commodities_organization" ON "commodities"("organization_id");

CREATE INDEX "idx_commodities_created_updated" ON "commodities"("created_at", "updated_at");

COMMENT ON TABLE "commodities" IS 'Stores information about commodities';

ALTER TABLE "commodities"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_commodities_search_vector ON "commodities" USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION commodities_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.name, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS commodities_search_update ON "commodities";

CREATE TRIGGER commodities_search_update
    BEFORE INSERT OR UPDATE ON "commodities"
    FOR EACH ROW
    EXECUTE FUNCTION commodities_search_trigger();

--bun:split
UPDATE
    "commodities"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(name, '')), 'A') || setweight(to_tsvector('english', COALESCE(description, '')), 'B');

