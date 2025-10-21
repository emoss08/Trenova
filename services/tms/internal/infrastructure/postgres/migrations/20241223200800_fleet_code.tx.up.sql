CREATE TABLE IF NOT EXISTS "fleet_codes"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "code" varchar(10) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "manager_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "description" text,
    "revenue_goal" numeric(10, 2),
    "deadhead_goal" numeric(10, 2),
    "mileage_goal" numeric(10, 2),
    "color" varchar(10),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_fleet_codes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_fleet_codes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fleet_codes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_fleet_codes_manager" FOREIGN KEY ("manager_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split
-- Indexes for fleet_codes table
CREATE INDEX "idx_fleet_codes_manager" ON "fleet_codes"("manager_id")
WHERE
    manager_id IS NOT NULL;

CREATE INDEX "idx_fleet_codes_color" ON "fleet_codes"("color");

CREATE INDEX "idx_fleet_codes_created_updated" ON "fleet_codes"("created_at", "updated_at");

COMMENT ON TABLE "fleet_codes" IS 'Stores information about fleet codes';

--bun:split
--alter the workers table to add the fleet_code_id column
ALTER TABLE "workers"
    ADD COLUMN "fleet_code_id" varchar(100);

--add the foreign key constraint
ALTER TABLE "workers"
    ADD CONSTRAINT "fk_workers_fleet_code" FOREIGN KEY ("fleet_code_id", "organization_id", "business_unit_id") REFERENCES "fleet_codes"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

-- Search Vector and Relationship Indexes SQL for FleetCode
-- Generated for optimal query performance including relationships
-- 1. Search Vector Column
ALTER TABLE "fleet_codes"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
-- 2. Search Vector Index (GIN for full-text search)
CREATE INDEX IF NOT EXISTS idx_fleet_codes_search_vector ON "fleet_codes" USING GIN(search_vector);

-- Search Vector and Relationship Indexes SQL for FleetCode
-- Generated for optimal query performance including relationships
-- 1. Search Vector Column
ALTER TABLE "fleet_codes"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
-- 2. Search Vector Index (GIN for full-text search)
CREATE INDEX IF NOT EXISTS idx_fleet_codes_search_vector ON "fleet_codes" USING GIN(search_vector);

--bun:split
-- 4. Search Vector Trigger Function
CREATE OR REPLACE FUNCTION fleet_codes_search_trigger()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('english', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('english', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
-- 5. Search Vector Trigger
DROP TRIGGER IF EXISTS fleet_codes_search_update ON "fleet_codes";

CREATE TRIGGER fleet_codes_search_update
    BEFORE INSERT OR UPDATE ON "fleet_codes"
    FOR EACH ROW
    EXECUTE FUNCTION fleet_codes_search_trigger();

--bun:split
-- 6. Update Existing Rows
UPDATE
    "fleet_codes"
SET
    search_vector = setweight(to_tsvector('english', COALESCE(code, '')), 'A') || setweight(to_tsvector('english', COALESCE(description, '')), 'B');

