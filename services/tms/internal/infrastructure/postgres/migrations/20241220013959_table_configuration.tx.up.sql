--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE configuration_visibility_enum AS ENUM(
    'Private',
    'Public',
    'Shared'
);

--bun:split
CREATE TABLE IF NOT EXISTS "table_configurations"(
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    "name" varchar(255) NOT NULL,
    "description" text,
    "resource" varchar(100) NOT NULL,
    "table_config" jsonb NOT NULL,
    "visibility" configuration_visibility_enum NOT NULL DEFAULT 'Private',
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_table_configurations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_table_configurations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_configurations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_configurations_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_table_config_format" CHECK (jsonb_typeof(table_config) = 'object')
);

CREATE UNIQUE INDEX "idx_table_configurations_default" ON "table_configurations"("user_id", "resource", "is_default")
WHERE
    "is_default" = TRUE;

CREATE INDEX "idx_table_configurations_business_unit" ON "table_configurations"("business_unit_id");

CREATE INDEX "idx_table_configurations_organization" ON "table_configurations"("organization_id");

CREATE INDEX "idx_table_configurations_user_id" ON "table_configurations"("user_id");

CREATE INDEX "idx_table_configurations_resource" ON "table_configurations"("resource");

CREATE INDEX "idx_table_configurations_visibility" ON "table_configurations"("visibility");

CREATE INDEX "idx_table_configurations_created_updated" ON "table_configurations"("created_at", "updated_at");

--bun:split
ALTER TABLE "table_configurations"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (
        setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') ||
        setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')
    ) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS idx_table_configurations_search ON table_configurations USING GIN(search_vector);

--bun:split
COMMENT ON TABLE table_configurations IS 'Stores saved table filter configurations for data tables';

--bun:split
CREATE TABLE IF NOT EXISTS "table_configuration_shares"(
    "id" varchar(100) NOT NULL,
    "configuration_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shared_with_id" varchar(100) NOT NULL,
    "share_type" varchar(20) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    CONSTRAINT "pk_table_configuration_shares" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_configuration_shares_config" FOREIGN KEY ("configuration_id", "organization_id", "business_unit_id") REFERENCES "table_configurations"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_configuration_shares" UNIQUE ("configuration_id", "shared_with_id", "share_type")
);

--bun:split
CREATE INDEX "idx_configuration_shares_config" ON "table_configuration_shares"("configuration_id", "organization_id");

CREATE INDEX "idx_configuration_shares_shared_with" ON "table_configuration_shares"("shared_with_id");

CREATE INDEX "idx_configuration_shares_type" ON "table_configuration_shares"("share_type");

CREATE INDEX "idx_configuration_shares_created" ON "table_configuration_shares"("created_at");

COMMENT ON TABLE table_configuration_shares IS 'Stores sharing information for table configurations';
