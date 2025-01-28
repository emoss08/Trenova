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
    "postal_code" varchar(10),
    "auto_mark_ready_to_bill" boolean NOT NULL DEFAULT FALSE,
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

COMMENT ON TABLE "customers" IS 'Stores information about customers';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "customer_id" varchar(100) NOT NULL;

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE;

