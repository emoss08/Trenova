--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TABLE IF NOT EXISTS "shipment_types"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "code" varchar(10) NOT NULL,
    "description" text,
    "color" varchar(10),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_shipment_types" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_shipment_types_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_shipment_types_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX "idx_shipment_types_code" ON "shipment_types"(lower("code"), "organization_id");

CREATE INDEX "idx_shipment_types_business_unit" ON "shipment_types"("business_unit_id");

CREATE INDEX "idx_shipment_types_organization" ON "shipment_types"("organization_id");

CREATE INDEX "idx_shipment_types_created_updated" ON "shipment_types"("created_at", "updated_at");

COMMENT ON TABLE "shipment_types" IS 'Stores information about shipment types';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "shipment_type_id" varchar(100);

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_shipment_type" FOREIGN KEY ("shipment_type_id", "business_unit_id", "organization_id") REFERENCES "shipment_types"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;

--bun:split
ALTER TABLE "shipment_types"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipment_types_search ON shipment_types USING GIN(search_vector);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipment_types_trgm_code ON shipment_types USING gin(code gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipment_types_trgm_description ON shipment_types USING gin(description gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS idx_shipment_types_trgm_code_description ON shipment_types USING gin((code || ' ' || description) gin_trgm_ops);
