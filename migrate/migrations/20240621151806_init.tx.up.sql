-- COPYRIGHT(c) 2024 Trenova
--
-- This file is part of Trenova.
--
-- The Trenova software is licensed under the Business Source License 1.1. You are granted the right
-- to copy, modify, and redistribute the software, but only for non-production use or with a total
-- of less than three server instances. Starting from the Change Date (November 16, 2026), the
-- software will be made available under version 2 or later of the GNU General Public License.
-- If you use the software in violation of this license, your rights under the license will be
-- terminated automatically. The software is provided "as is," and the Licensor disclaims all
-- warranties and conditions. If you use this license's text or the "Business Source License" name
-- and trademark, you must comply with the Licensor's covenants, which include specifying the
-- Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
-- Grant, and not modifying the license in any other way.


CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

--bun:split

CREATE EXTENSION IF NOT EXISTS pg_trgm;

--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'status_enum') THEN
            CREATE TYPE status_enum AS ENUM ('Active', 'Inactive');
        END IF;
    END
$$;


--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'database_action_enum') THEN
            CREATE TYPE database_action_enum AS ENUM ('Insert', 'Update', 'Delete', 'All');
        END IF;
    END
$$;

--bun:split

DO
$$
    BEGIN
        IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'delivery_method_enum') THEN
            CREATE TYPE delivery_method_enum AS ENUM ('Email', 'Local', 'Api', 'Sms');
        END IF;
    END
$$;

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "us_states"
(
    "created_at"   TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"   TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"           uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "name"         VARCHAR     NOT NULL,
    "abbreviation" VARCHAR     NOT NULL,
    "country_name" VARCHAR     NOT NULL,
    "country_iso3" VARCHAR     NOT NULL DEFAULT 'USA',
    PRIMARY KEY ("id")
);

--bun:split

CREATE INDEX idx_us_states_name ON us_states USING gin (name gin_trgm_ops);
CREATE INDEX idx_us_states_created_at ON us_states (created_at);

--bun:split

COMMENT ON COLUMN us_states.id IS 'Unique identifier for the US state, generated as a UUID';
COMMENT ON COLUMN us_states.name IS 'The name of the US state';
COMMENT ON COLUMN us_states.abbreviation IS 'The abbreviation of the US state';
COMMENT ON COLUMN us_states.country_name IS 'The name of the country that the US state belongs to';
COMMENT ON COLUMN us_states.country_iso3 IS 'The ISO3 code of the country that the US state belongs to';

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "business_units"
(
    "created_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "updated_at"    TIMESTAMPTZ  NOT NULL DEFAULT current_timestamp,
    "id"            uuid         NOT NULL DEFAULT uuid_generate_v4(),
    "name"          VARCHAR(100) NOT NULL,
    "address_line1" VARCHAR,
    "address_line2" VARCHAR,
    "city"          VARCHAR,
    "state_id"      uuid,
    "postal_code"   VARCHAR,
    "phone_number"  VARCHAR(15),
    "contact_name"  VARCHAR,
    "contact_email" VARCHAR,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("state_id") REFERENCES "us_states" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

--================================================
--bun:split
CREATE INDEX idx_business_units_name ON business_units USING gin (name gin_trgm_ops);
CREATE INDEX idx_business_units_created_at ON business_units (created_at);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "organizations"
(
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "business_unit_id" uuid,
    "name"             VARCHAR     NOT NULL,
    "scac_code"        VARCHAR     NOT NULL,
    "dot_number"       VARCHAR,
    "logo_url"         VARCHAR,
    "org_type"         VARCHAR     NOT NULL DEFAULT 'Asset',
    "address_line_1"   VARCHAR,
    "address_line_2"   VARCHAR,
    "city"             VARCHAR,
    "version"          BIGINT      NOT NULL,
    "state_id"         uuid        NOT NULL,
    "postal_code"      VARCHAR,
    "timezone"         VARCHAR     NOT NULL DEFAULT 'America/New_York',
    PRIMARY KEY ("id"),
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--================================================
--bun:split

CREATE INDEX idx_organizations_org_bu ON organizations (business_unit_id);
CREATE INDEX idx_organizations_name ON organizations USING gin (name gin_trgm_ops);
CREATE INDEX idx_organizations_created_at ON organizations (created_at);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "resources"
(
    "created_at"  TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"  TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"          uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "type"        VARCHAR,
    "description" VARCHAR,
    PRIMARY KEY ("id")
);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "permissions"
(
    "created_at"        TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"        TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"                uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "codename"          VARCHAR,
    "action"            VARCHAR,
    "label"             VARCHAR,
    "read_description"  VARCHAR,
    "write_description" VARCHAR,
    "resource_id"       uuid        NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("resource_id") REFERENCES "resources" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

--================================================
--bun:split

CREATE INDEX idx_permissions_codename ON permissions USING gin (codename gin_trgm_ops);
CREATE INDEX idx_permissions_created_at ON permissions (created_at);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "roles"
(
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "name"             VARCHAR     NOT NULL,
    "description"      TEXT,
    "color"            VARCHAR,
    "version"          BIGINT      NOT NULL,
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

--================================================
--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "roles_name_organization_id_unq" ON "roles" (LOWER("name"), organization_id);
CREATE INDEX idx_roles_org_bu ON roles (organization_id, business_unit_id);
CREATE INDEX idx_roles_name ON roles USING gin (name gin_trgm_ops);
CREATE INDEX idx_roles_description ON roles USING GIN (description gin_trgm_ops);
CREATE INDEX idx_roles_created_at ON roles (created_at);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "role_permissions"
(
    "role_id"       uuid NOT NULL,
    "permission_id" uuid NOT NULL,
    PRIMARY KEY ("role_id", "permission_id"),
    FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    FOREIGN KEY ("permission_id") REFERENCES "permissions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "users"
(
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "name"             VARCHAR,
    "username"         VARCHAR     NOT NULL,
    "password"         VARCHAR,
    "email"            VARCHAR     NOT NULL,
    "timezone"         VARCHAR     NOT NULL,
    "profile_pic_url"  VARCHAR,
    "thumbnail_url"    VARCHAR,
    "version"          BIGINT      NOT NULL,
    "phone_number"     VARCHAR,
    "is_admin"         BOOLEAN              DEFAULT false,
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    PRIMARY KEY ("id"),
    UNIQUE ("email"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--================================================
--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "users_email_unq" ON "users" (LOWER("email"));
CREATE UNIQUE INDEX IF NOT EXISTS "users_username_organization_id_unq" ON "users" (LOWER("username"), organization_id);
CREATE INDEX idx_users_org_bu ON users (organization_id, business_unit_id);
CREATE INDEX idx_users_name ON users USING gin (name gin_trgm_ops);
CREATE INDEX idx_users_created_at ON users (created_at);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "user_favorites"
(
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "page_link"        VARCHAR     NOT NULL,
    "user_id"          uuid        NOT NULL,
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "user_notifications"
(
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "is_read"          BOOLEAN     NOT NULL DEFAULT false,
    "title"            VARCHAR     NOT NULL,
    "description"      TEXT        NOT NULL,
    "action_url"       VARCHAR,
    "user_id"          uuid        NOT NULL,
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

--================================================
--bun:split

CREATE INDEX idx_user_favorites_org_bu ON user_favorites (organization_id, business_unit_id);
CREATE INDEX idx_user_favorites_page_link ON user_favorites USING gin (page_link gin_trgm_ops);
CREATE INDEX idx_user_favorites_created_at ON user_favorites (created_at);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "user_roles"
(
    "created_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at" TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "user_id"    uuid        NOT NULL,
    "role_id"    uuid        NOT NULL,
    PRIMARY KEY ("user_id", "role_id"),
    FOREIGN KEY ("role_id") REFERENCES "roles" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
    FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "fleet_codes"
(
    "created_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ NOT NULL DEFAULT current_timestamp,
    "id"               uuid        NOT NULL DEFAULT uuid_generate_v4(),
    "status"           status_enum NOT NULL DEFAULT 'Active',
    "code"             VARCHAR(10) NOT NULL,
    "description"      VARCHAR(100),
    "revenue_goal"     numeric(10, 2),
    "deadhead_goal"    numeric(10, 2),
    "mileage_goal"     numeric(10, 2),
    "color"            VARCHAR(10),
    "version"          BIGINT      NOT NULL,
    "manager_id"       uuid,
    "business_unit_id" uuid        NOT NULL,
    "organization_id"  uuid        NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("manager_id") REFERENCES users ("id") ON UPDATE NO ACTION ON DELETE RESTRICT
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "fleet_codes_code_organization_id_unq" ON "fleet_codes" (LOWER("code"), organization_id);
CREATE INDEX idx_fleet_codes_org_bu ON fleet_codes (organization_id, business_unit_id);
CREATE INDEX idx_fleet_codes_code ON fleet_codes USING gin (code gin_trgm_ops);
CREATE INDEX idx_fleet_codes_description ON fleet_codes USING GIN (description gin_trgm_ops);
CREATE INDEX idx_fleet_codes_created_at ON fleet_codes (created_at);

--bun:split

COMMENT ON COLUMN fleet_codes.id IS 'Unique identifier for the fleet code, generated as a UUID';
COMMENT ON COLUMN fleet_codes.status IS 'The current status of the fleet code, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN fleet_codes.code IS 'A short, unique code for identifying the fleet code, limited to 10 characters';
COMMENT ON COLUMN fleet_codes.description IS 'A description of the fleet code, limited to 100 characters';
COMMENT ON COLUMN fleet_codes.revenue_goal IS 'The revenue goal for the fleet code';
COMMENT ON COLUMN fleet_codes.deadhead_goal IS 'The deadhead goal for the fleet code';
COMMENT ON COLUMN fleet_codes.mileage_goal IS 'The mileage goal for the fleet code';
COMMENT ON COLUMN fleet_codes.color IS 'The color associated with the fleet code, limited to 10 characters';
COMMENT ON COLUMN fleet_codes.manager_id IS 'Foreign key referencing the user that manages the fleet code';
COMMENT ON COLUMN fleet_codes.business_unit_id IS 'Foreign key referencing the business unit that this fleet code belongs to';
COMMENT ON COLUMN fleet_codes.organization_id IS 'Foreign key referencing the organization that this fleet code belongs to';
COMMENT ON COLUMN fleet_codes.created_at IS 'Timestamp of when the fleet code was created, defaults to the current timestamp';
COMMENT ON COLUMN fleet_codes.updated_at IS 'Timestamp of the last update to the fleet code, defaults to the current timestamp';
