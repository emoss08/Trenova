-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20241211015659_user.tx.up.sql

CREATE TABLE IF NOT EXISTS "users"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "current_organization_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "name" TEXT NOT NULL,
    "username" TEXT NOT NULL,
    "password" TEXT NOT NULL,
    "email_address" TEXT NOT NULL,
    "timezone" TEXT NOT NULL,
    "time_format" TEXT NOT NULL DEFAULT '12-hour',
    "profile_pic_url" TEXT,
    "thumbnail_url" TEXT,
    "is_locked" INTEGER NOT NULL DEFAULT 0,
    "must_change_password" INTEGER NOT NULL DEFAULT 0,
    "last_login_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_users_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_users_current_organization" FOREIGN KEY ("current_organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_users_email_address" ON "users" (lower("email_address"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_users_username" ON "users" (lower("username"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_users_business_unit" ON "users" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_users_current_organization" ON "users" ("current_organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_users_bu_org_id" ON "users" ("business_unit_id", "current_organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_users_bu_org_username" ON "users" ("business_unit_id", "current_organization_id", "username");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_users_status" ON "users" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_users_created_updated" ON "users" ("created_at", "updated_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_users_org_bu_no_system" ON "users" ("current_organization_id", "business_unit_id")WHERE
    "username" != 'admin';

--bun:split

CREATE TABLE IF NOT EXISTS "user_organizations"(
    "user_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("user_id", "organization_id"),
    CONSTRAINT "fk_user_organizations_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_organizations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_user_organizations_user" ON "user_organizations" ("user_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_user_organizations_organization" ON "user_organizations" ("organization_id");
