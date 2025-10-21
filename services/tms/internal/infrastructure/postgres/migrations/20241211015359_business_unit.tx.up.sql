--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

CREATE TYPE "business_unit_status_enum" AS ENUM (
    'Active',
    'Inactive',
    'Suspended',
    'Pending'
);

CREATE TABLE IF NOT EXISTS "business_units" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "parent_business_unit_id" varchar(100),
    "state_id" varchar(100),
    -- Core fields
    "name" varchar(100) NOT NULL,
    "code" varchar(10) NOT NULL,
    "description" text,
    "status" business_unit_status_enum NOT NULL DEFAULT 'Active',
    "primary_contact" varchar(100),
    "primary_email" varchar(255),
    "primary_phone" varchar(20),
    "address_line1" varchar(100),
    "address_line2" varchar(100),
    "city" varchar(100),
    "postal_code" us_postal_code NOT NULL,
    "timezone" varchar(50) NOT NULL DEFAULT 'America/New_York',
    "locale" varchar(10) NOT NULL DEFAULT 'en-US',
    "tax_id" varchar(50),
    -- Metadata & Versioning
    "metadata" jsonb DEFAULT '{}' ::jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_business_units_state" FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "fk_business_units_parent" FOREIGN KEY ("parent_business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_parent_not_self" CHECK ("id" != "parent_business_unit_id"),
    CONSTRAINT "check_metadata_format" CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX "idx_business_units_name" ON "business_units" (lower("name"));

CREATE UNIQUE INDEX "idx_business_units_code" ON "business_units" (lower("code"));

CREATE INDEX "idx_business_units_status" ON "business_units" ("status");

CREATE INDEX "idx_business_units_parent" ON "business_units" ("parent_business_unit_id");

CREATE INDEX "idx_business_units_created_updated" ON "business_units" ("created_at", "updated_at");

CREATE INDEX "idx_business_units_metadata" ON "business_units" USING gin ("metadata");

COMMENT ON TABLE business_units IS 'Stores information about business units in a hierarchical structure';

