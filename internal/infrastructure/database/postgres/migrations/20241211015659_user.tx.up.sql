-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

CREATE TYPE "status_enum" AS ENUM(
    'Active',
    'Inactive'
);

CREATE TYPE "time_format_enum" AS ENUM(
    '12-hour',
    '24-hour'
);

--bun:split
CREATE TABLE IF NOT EXISTS "users"(
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "current_organization_id" varchar(100) NOT NULL,
    -- Core fields
    "status" status_enum NOT NULL DEFAULT 'Active',
    "name" varchar(255) NOT NULL,
    "username" varchar(20) NOT NULL,
    "password" varchar(255) NOT NULL,
    "email_address" varchar(255) NOT NULL,
    "timezone" varchar(50) NOT NULL,
    "time_format" time_format_enum NOT NULL DEFAULT '12-hour',
    "profile_pic_url" varchar(255),
    "thumbnail_url" varchar(255),
    "is_locked" boolean NOT NULL DEFAULT FALSE,
    "must_change_password" boolean NOT NULL DEFAULT FALSE,
    "last_login_at" bigint,
    -- Metadata and versioning
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_users_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_users_current_organization" FOREIGN KEY ("current_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "idx_users_email_address" ON "users"(lower("email_address"));

CREATE UNIQUE INDEX IF NOT EXISTS "idx_users_username" ON "users"(lower("username"));

CREATE INDEX IF NOT EXISTS "idx_users_business_unit" ON "users"("business_unit_id");

CREATE INDEX IF NOT EXISTS "idx_users_current_organization" ON "users"("current_organization_id");

CREATE INDEX IF NOT EXISTS "idx_users_status" ON "users"("status");

CREATE INDEX IF NOT EXISTS "idx_users_created_updated" ON "users"("created_at", "updated_at");

COMMENT ON TABLE users IS 'Stores information about users';

--bun:split
-- Function to validate current_organization belongs to business_unit
CREATE OR REPLACE FUNCTION check_user_organization_business_unit()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF NEW.current_organization_id IS NOT NULL THEN
        IF NOT EXISTS(
            SELECT
                1
            FROM
                organizations o
            WHERE
                o.id = NEW.current_organization_id
                AND o.business_unit_id = NEW.business_unit_id) THEN
        RAISE EXCEPTION 'Current organization must belong to the same business unit as the user';
    END IF;
END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

-- Trigger to enforce business unit constraint on users
CREATE TRIGGER trigger_check_user_organization_business_unit
    BEFORE INSERT OR UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION check_user_organization_business_unit();

--bun:split
-- User Organizations mapping table
CREATE TABLE IF NOT EXISTS "user_organizations"(
    "user_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    PRIMARY KEY ("user_id", "organization_id"),
    CONSTRAINT "fk_user_organizations_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_user_organizations_user" ON "user_organizations"("user_id");

CREATE INDEX IF NOT EXISTS "idx_user_organizations_organization" ON "user_organizations"("organization_id");

COMMENT ON TABLE user_organizations IS 'Mapping table for users and organizations';

--bun:split
-- Function to validate user_organization business unit match
CREATE OR REPLACE FUNCTION check_user_organization_mapping()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF NOT EXISTS(
        SELECT
            1
        FROM
            users u
            JOIN organizations o ON o.id = NEW.organization_id
        WHERE
            u.id = NEW.user_id
            AND u.business_unit_id = o.business_unit_id) THEN
    RAISE EXCEPTION 'User and organization must belong to the same business unit';
END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

-- Trigger to enforce business unit constraint on user_organizations
CREATE TRIGGER trigger_check_user_organization_mapping
    BEFORE INSERT OR UPDATE ON user_organizations
    FOR EACH ROW
    EXECUTE FUNCTION check_user_organization_mapping();

ALTER TABLE users
    ALTER COLUMN status SET STATISTICS 1000;

ALTER TABLE users
    ALTER COLUMN business_unit_id SET STATISTICS 1000;

ALTER TABLE users
    ALTER COLUMN current_organization_id SET STATISTICS 1000;

