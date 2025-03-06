CREATE TYPE "org_type_enum" AS ENUM (
    'Carrier',
    'Brokerage',
    'BrokerageCarrier'
);

CREATE TABLE IF NOT EXISTS "organizations" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    -- Core fields
    "name" varchar(100) NOT NULL,
    "scac_code" varchar(4) NOT NULL,
    "dot_number" varchar(8) NOT NULL,
    "logo_url" varchar(255),
    "org_type" org_type_enum NOT NULL DEFAULT 'Carrier',
    "bucket_name" varchar(63) NOT NULL,
    "address_line1" varchar(150) NOT NULL,
    "address_line2" varchar(150),
    "city" varchar(100) NOT NULL,
    "postal_code" varchar(20),
    "timezone" varchar(100) NOT NULL DEFAULT 'America/New_York',
    "tax_id" varchar(50),
    -- Metadata and versioning
    "metadata" jsonb DEFAULT '{}' ::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_organizations_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_organizations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    -- ! CONSTRAINT "check_scac_format" CHECK (scac_code ~ '^[A-Z]{4}$'), we will check this in the business logic
    -- ! CONSTRAINT "check_dot_format" CHECK (dot_number ~ '^\d{1,8}$'), we will check this in the business logic
    CONSTRAINT "check_metadata_format" CHECK (jsonb_typeof(metadata) = 'object')
);

-- Indexes
CREATE UNIQUE INDEX "idx_organizations_name_business_unit" ON "organizations" (lower("name"), "business_unit_id");

CREATE UNIQUE INDEX "idx_organizations_scac_business_unit" ON "organizations" ("scac_code", "business_unit_id");

CREATE UNIQUE INDEX "idx_organizations_dot_business_unit" ON "organizations" ("dot_number", "business_unit_id");

CREATE INDEX "idx_organizations_business_unit" ON "organizations" ("business_unit_id");

CREATE INDEX "idx_organizations_state" ON "organizations" ("state_id");

CREATE INDEX "idx_organizations_type" ON "organizations" ("org_type");

CREATE INDEX "idx_organizations_created_updated" ON "organizations" ("created_at", "updated_at");

CREATE INDEX "idx_organizations_metadata" ON "organizations" USING gin ("metadata");

COMMENT ON TABLE organizations IS 'Stores information about organizations within business units';

