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

--================================================
--bun:split

CREATE TABLE
    IF NOT EXISTS "table_change_alerts"
(
    "created_at"       TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    "updated_at"       TIMESTAMPTZ          NOT NULL DEFAULT current_timestamp,
    "id"               uuid                 NOT NULL DEFAULT uuid_generate_v4(),
    "status"           status_enum          NOT NULL DEFAULT 'Active',
    "name"             VARCHAR(50)          NOT NULL,
    "database_action"  database_action_enum NOT NULL,
    "topic_name"       VARCHAR(200)         NOT NULL,
    "description"      TEXT,
    "custom_subject"   VARCHAR,
    "delivery_method"  delivery_method_enum NOT NULL DEFAULT 'Email',
    "email_recipients" TEXT,
    "effective_date"   date,
    "expiration_date"  date,
    "business_unit_id" uuid                 NOT NULL,
    "organization_id"  uuid                 NOT NULL,
    PRIMARY KEY ("id"),
    FOREIGN KEY ("organization_id") REFERENCES organizations ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    FOREIGN KEY ("business_unit_id") REFERENCES business_units ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "table_change_alerts_name_organization_id_unq" ON "table_change_alerts" (LOWER("name"), organization_id);
CREATE INDEX idx_table_change_alerts_org_bu ON table_change_alerts (organization_id, business_unit_id);
CREATE INDEX idx_table_change_alerts_code ON table_change_alerts USING gin (name gin_trgm_ops);
CREATE INDEX idx_table_change_alerts_created_at ON table_change_alerts (created_at);

--bun:split

COMMENT ON COLUMN table_change_alerts.id IS 'Unique identifier for the table change alert, generated as a UUID';
COMMENT ON COLUMN table_change_alerts.status IS 'The current status of the table change alert, using the status_enum (e.g., Active, Inactive)';
COMMENT ON COLUMN table_change_alerts.name IS 'A short, unique name for identifying the table change alert, limited to 50 characters';
COMMENT ON COLUMN table_change_alerts.database_action IS 'The database action that triggers the table change alert, using the database_action_enum (e.g., Insert, Update, Delete, All)';
COMMENT ON COLUMN table_change_alerts.topic_name IS 'The name of the topic that the table change alert is associated with, limited to 200 characters';
COMMENT ON COLUMN table_change_alerts.description IS 'A description of the table change alert';
COMMENT ON COLUMN table_change_alerts.custom_subject IS 'A custom subject for the table change alert, limited to 200 characters';
COMMENT ON COLUMN table_change_alerts.delivery_method IS 'The delivery method for the table change alert, using the delivery_method_enum (e.g., Email, Local, Api, Sms)';
COMMENT ON COLUMN table_change_alerts.email_recipients IS 'A list of email recipients for the table change alert';
COMMENT ON COLUMN table_change_alerts.effective_date IS 'The effective date for the table change alert';
COMMENT ON COLUMN table_change_alerts.expiration_date IS 'The expiration date for the table change alert';
COMMENT ON COLUMN table_change_alerts.business_unit_id IS 'Foreign key referencing the business unit that this table change alert belongs to';
COMMENT ON COLUMN table_change_alerts.organization_id IS 'Foreign key referencing the organization that this table change alert belongs to';
