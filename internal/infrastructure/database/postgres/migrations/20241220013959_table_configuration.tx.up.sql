-- Configuration visibility type enum with descriptions
CREATE TYPE configuration_visibility_enum AS ENUM (
    'Private', -- Only visible to the creator
    'Public', -- Visible to everyone in the organization
    'Shared' -- Visible to specific users/roles
);

--bun:split
CREATE TABLE IF NOT EXISTS "table_configurations" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    -- Core Fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" varchar(255) NOT NULL,
    "description" text,
    "table_identifier" varchar(100) NOT NULL,
    "filter_config" jsonb NOT NULL,
    "visibility" configuration_visibility_enum NOT NULL DEFAULT 'Private',
    "is_default" boolean NOT NULL DEFAULT FALSE,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_table_configurations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_table_configurations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_configurations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_table_configurations_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "check_filter_config_format" CHECK (jsonb_typeof(filter_config) = 'object')
);

--bun:split
-- Indexes for table_configurations
-- Ensure that the name is unique for the given organization, and table identifier
CREATE UNIQUE INDEX "idx_table_configurations_name" ON "table_configurations" ("organization_id", "table_identifier", LOWER("name"));

CREATE INDEX "idx_table_configurations_business_unit" ON "table_configurations" ("business_unit_id");

CREATE INDEX "idx_table_configurations_organization" ON "table_configurations" ("organization_id");

CREATE INDEX "idx_table_configurations_user_id" ON "table_configurations" ("user_id");

CREATE INDEX "idx_table_configurations_table_id" ON "table_configurations" ("table_identifier");

CREATE INDEX "idx_table_configurations_visibility" ON "table_configurations" ("visibility");

CREATE INDEX "idx_table_configurations_default" ON "table_configurations" ("is_default")
WHERE
    is_default = TRUE;

CREATE INDEX "idx_table_configurations_created_updated" ON "table_configurations" ("created_at", "updated_at");

COMMENT ON TABLE table_configurations IS 'Stores saved table filter configurations for data tables';

--bun:split
CREATE TABLE IF NOT EXISTS "table_configuration_shares" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "configuration_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shared_with_id" varchar(100) NOT NULL,
    -- Core Fields
    "share_type" varchar(20) NOT NULL, -- 'user', 'role', 'team'
    -- Metadata
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_table_configuration_shares" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_configuration_shares_config" FOREIGN KEY ("configuration_id", "organization_id", "business_unit_id") REFERENCES "table_configurations" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "uq_configuration_shares" UNIQUE ("configuration_id", "shared_with_id", "share_type")
);

--bun:split
-- Indexes for table_configuration_shares
CREATE INDEX "idx_configuration_shares_config" ON "table_configuration_shares" ("configuration_id", "organization_id");

CREATE INDEX "idx_configuration_shares_shared_with" ON "table_configuration_shares" ("shared_with_id");

CREATE INDEX "idx_configuration_shares_type" ON "table_configuration_shares" ("share_type");

CREATE INDEX "idx_configuration_shares_created" ON "table_configuration_shares" ("created_at");

COMMENT ON TABLE table_configuration_shares IS 'Stores sharing information for table configurations';

