--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "service_types"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "description" text,
    "color" varchar(10),
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_service_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_service_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_service_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- Indexes for service_types table
CREATE UNIQUE INDEX "idx_service_types_code" ON "service_types"(lower("code"), "organization_id");

CREATE INDEX "idx_service_types_business_unit" ON "service_types"("business_unit_id");

CREATE INDEX "idx_service_types_organization" ON "service_types"("organization_id");

CREATE INDEX "idx_service_types_created_updated" ON "service_types"("created_at", "updated_at");

COMMENT ON TABLE "service_types" IS 'Stores information about service types';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "service_type_id" varchar(100) NOT NULL;

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_service_type" FOREIGN KEY ("service_type_id", "business_unit_id", "organization_id") REFERENCES "service_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "service_types"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_service_types_search ON service_types USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION service_types_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.description, '')), 'B');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS service_types_search_vector_trigger ON service_types;

--bun:split
CREATE TRIGGER service_types_search_vector_trigger
    BEFORE INSERT OR UPDATE ON service_types
    FOR EACH ROW
    EXECUTE FUNCTION service_types_search_vector_update();

--bun:split
CREATE INDEX IF NOT EXISTS idx_service_types_active ON service_types(created_at DESC)
WHERE
    status != 'Inactive';

--bun:split
ALTER TABLE service_types
    ALTER COLUMN status SET STATISTICS 1000;

--bun:split
ALTER TABLE service_types
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE service_types
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_service_types_trgm_code ON service_types USING gin(code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_service_types_trgm_description ON service_types USING gin(description gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_service_types_trgm_code_description ON service_types USING gin((code || ' ' || description) gin_trgm_ops);

