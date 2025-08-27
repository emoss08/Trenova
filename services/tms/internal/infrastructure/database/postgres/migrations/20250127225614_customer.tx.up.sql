--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TABLE IF NOT EXISTS "customers"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "name" text,
    "address_line_1" varchar(150),
    "address_line_2" varchar(150),
    "city" varchar(100),
    "postal_code" us_postal_code NOT NULL,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_customers" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_customers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_customers_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split
-- Indexes for customers table
CREATE UNIQUE INDEX "idx_customers_code" ON "customers"(lower("code"), "organization_id");

CREATE INDEX "idx_customers_name" ON "customers"("name");

CREATE INDEX "idx_customers_business_unit_organization" ON "customers"("business_unit_id", "organization_id");

CREATE INDEX "idx_customers_created_updated" ON "customers"("created_at", "updated_at");

CREATE INDEX "idx_customers_status" ON "customers"("status");

COMMENT ON TABLE "customers" IS 'Stores information about customers';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "customer_id" varchar(100) NOT NULL;

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE;

--bun:split
ALTER TABLE "customers"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS idx_customers_search ON customers USING GIN(search_vector);

--bun:split
CREATE OR REPLACE FUNCTION customers_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.code, '')), 'A') || setweight(to_tsvector('simple', COALESCE(NEW.name, '')), 'B');
    -- Auto-update timestamps
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS customers_search_vector_trigger ON customers;

--bun:split
CREATE TRIGGER customers_search_vector_trigger
    BEFORE INSERT OR UPDATE ON customers
    FOR EACH ROW
    EXECUTE FUNCTION customers_search_vector_update();

--bun:split
ALTER TABLE customers
    ALTER COLUMN status SET STATISTICS 1000;

--bun:split
ALTER TABLE customers
    ALTER COLUMN organization_id SET STATISTICS 1000;

--bun:split
ALTER TABLE customers
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS idx_customers_trgm_code ON customers USING gin(code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_customers_trgm_name ON customers USING gin(name gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_customers_trgm_code_name ON customers USING gin((code || ' ' || name) gin_trgm_ops);

