CREATE TABLE IF NOT EXISTS "service_types"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(100) NOT NULL,
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
    ADD COLUMN "service_type_id" varchar(100) NOT NULL;

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_service_type" FOREIGN KEY ("service_type_id", "business_unit_id", "organization_id") REFERENCES "service_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

