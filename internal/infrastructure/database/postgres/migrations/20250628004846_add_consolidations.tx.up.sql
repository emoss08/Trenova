CREATE TYPE "consolidation_group_status_enum" AS ENUM(
    'New',
    'InProgress',
    'Completed',
    'Canceled'
);

--bun:split
CREATE TABLE IF NOT EXISTS "consolidation_groups"(
    "id" varchar(100) NOT NULL,
    "consolidation_number" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "status" consolidation_group_status_enum NOT NULL DEFAULT 'New',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_consolidation_groups" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_consolidation_groups_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_consolidation_groups_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "idx_consolidation_groups_consolidation_number" ON "consolidation_groups"("consolidation_number", "organization_id");

CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_status" ON "consolidation_groups"("status");

CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_business_unit" ON "consolidation_groups"("business_unit_id", "organization_id");

--bun:split
ALTER TABLE "consolidation_groups"
    ADD COLUMN IF NOT EXISTS search_vector tsvector;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_search" ON "consolidation_groups" USING GIN("search_vector");

--bun:split
CREATE OR REPLACE FUNCTION consolidation_groups_search_vector_update()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.search_vector := setweight(to_tsvector('simple', COALESCE(NEW.consolidation_number, '')), 'A') || setweight(to_tsvector('english', COALESCE(CAST(NEW.status AS text), '')), 'B');
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS consolidation_groups_search_vector_trigger ON "consolidation_groups";

--bun:split
CREATE TRIGGER consolidation_groups_search_vector_trigger
    BEFORE INSERT OR UPDATE ON "consolidation_groups"
    FOR EACH ROW
    EXECUTE FUNCTION consolidation_groups_search_vector_update();

--bun:split
ALTER TABLE "consolidation_groups"
    ALTER COLUMN "status" SET STATISTICS 1000;

--bun:split
ALTER TABLE "consolidation_groups"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "consolidation_groups"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_trgm_consolidation_number" ON "consolidation_groups" USING gin("consolidation_number" gin_trgm_ops);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_bu_org_status_created_at" ON "consolidation_groups"("business_unit_id", "organization_id", "status", "created_at" DESC);

CREATE INDEX IF NOT EXISTS "idx_consolidation_groups_bu_org_include" ON "consolidation_groups"("business_unit_id", "organization_id") INCLUDE ("status", "created_at", "consolidation_number");

--bun:split
COMMENT ON TABLE "consolidation_groups" IS 'Stores information about consolidation groups';

COMMENT ON COLUMN "consolidation_groups"."consolidation_number" IS 'Unique identifier for the consolidation group';

COMMENT ON COLUMN "consolidation_groups"."status" IS 'Status of the consolidation group';

--bun:split
ALTER TABLE "customers"
    ADD COLUMN "allow_consolidation" boolean NOT NULL DEFAULT TRUE;

ALTER TABLE "customers"
    ADD COLUMN "exclusive_consolidation" boolean NOT NULL DEFAULT FALSE;

ALTER TABLE "customers"
    ADD COLUMN "consolidation_priority" smallint NOT NULL DEFAULT 1;

ALTER TABLE "customers"
    ADD CONSTRAINT "ck_customers_consolidation_priority" CHECK ("consolidation_priority" >= 1 AND "consolidation_priority" <= 5);

--bun:split
COMMENT ON COLUMN "customers"."allow_consolidation" IS 'Whether this customer''s shipments are eligible for consolidation with other shipments';

COMMENT ON COLUMN "customers"."exclusive_consolidation" IS 'Whether this customer''s shipments can only be consolidated with other shipments from the same customer';

COMMENT ON COLUMN "customers"."consolidation_priority" IS 'Priority level for consolidation suggestions (1=low, 5=high)';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN "consolidation_group_id" varchar(100);

ALTER TABLE "shipments"
    ADD CONSTRAINT "fk_shipments_consolidation_group" FOREIGN KEY ("consolidation_group_id", "organization_id", "business_unit_id") REFERENCES "consolidation_groups"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE SET NULL;

COMMENT ON COLUMN "shipments"."consolidation_group_id" IS 'Simple consolidation group identifier - links shipments that should be coordinated together';

--bun:split
CREATE TABLE IF NOT EXISTS "consolidation_settings"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "max_pickup_distance" float NOT NULL DEFAULT 25,
    "max_delivery_distance" float NOT NULL DEFAULT 25,
    "max_route_detour" float NOT NULL DEFAULT 15,
    "max_time_window_gap" bigint NOT NULL DEFAULT 240,
    "min_time_buffer" bigint NOT NULL DEFAULT 30,
    "max_shipments_per_group" int NOT NULL DEFAULT 3,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP) ::bigint,
    -- Constraints
    CONSTRAINT "pk_consolidation_settings" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_consolidation_settings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_consolidation_settings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Ensure one consolidation group per organization
    CONSTRAINT "uq_consolidation_groups_organization" UNIQUE ("organization_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_consolidation_settings_business_unit" ON "consolidation_settings"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_consolidation_settings_created_at" ON "consolidation_settings"("created_at", "updated_at");

--bun:split
CREATE OR REPLACE FUNCTION consolidation_settings_update_timestamp()
    RETURNS TRIGGER
    AS $$
BEGIN
    NEW.updated_at := EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

--bun:split
DROP TRIGGER IF EXISTS consolidation_settings_update_timestamp_trigger ON "consolidation_settings";

--bun:split
CREATE TRIGGER consolidation_settings_update_timestamp_trigger
    BEFORE UPDATE ON "consolidation_settings"
    FOR EACH ROW
    EXECUTE FUNCTION consolidation_settings_update_timestamp();

--bun:split
ALTER TABLE "consolidation_settings"
    ALTER COLUMN "business_unit_id" SET STATISTICS 1000;

--bun:split
ALTER TABLE "consolidation_settings"
    ALTER COLUMN "organization_id" SET STATISTICS 1000;

--bun:split
COMMENT ON TABLE "consolidation_settings" IS 'Stores configuration for consolidation settings';

COMMENT ON COLUMN "consolidation_settings"."max_pickup_distance" IS 'Maximum distance in miles between pickup locations for shipments to be considered for consolidation';

COMMENT ON COLUMN "consolidation_settings"."max_delivery_distance" IS 'Maximum distance in miles between delivery locations for shipments to be considered for consolidation';

COMMENT ON COLUMN "consolidation_settings"."max_route_detour" IS 'Maximum percentage increase in total route distance that''s acceptable for consolidation';

COMMENT ON COLUMN "consolidation_settings"."max_time_window_gap" IS 'Maximum time gap in minutes between shipments'' planned pickup/delivery windows for consolidation';

COMMENT ON COLUMN "consolidation_settings"."min_time_buffer" IS 'Minimum time buffer in minutes required between stops when consolidating shipments';

COMMENT ON COLUMN "consolidation_settings"."max_shipments_per_group" IS 'Maximum number of shipments that can be consolidated into a single group';

