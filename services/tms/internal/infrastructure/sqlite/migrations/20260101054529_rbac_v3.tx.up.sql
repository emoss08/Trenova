-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260101054529_rbac_v3.tx.up.sql

DROP VIEW IF EXISTS user_effective_policies;

--bun:split

DROP TABLE IF EXISTS policy_templates;

--bun:split

DROP TABLE IF EXISTS permission_cache;

--bun:split

DROP TABLE IF EXISTS user_organization_roles;

--bun:split

DROP TABLE IF EXISTS roles;

--bun:split

DROP TABLE IF EXISTS policies;

--bun:split

CREATE TABLE IF NOT EXISTS roles(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "parent_role_ids" TEXT,
    "max_sensitivity" TEXT NOT NULL DEFAULT 'internal',
    "is_system" INTEGER NOT NULL DEFAULT 0,
    "created_by" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_roles" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_roles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_roles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_roles_created_by" FOREIGN KEY ("created_by") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uniq_roles_id" ON roles ("id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uniq_roles_name_org" ON roles (lower("name"), "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS resource_permissions(
    "id" TEXT NOT NULL,
    "role_id" TEXT NOT NULL,
    "resource" TEXT NOT NULL,
    "operations" TEXT NOT NULL,
    "data_scope" TEXT NOT NULL DEFAULT 'organization',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_resource_permissions" PRIMARY KEY ("id"),
    CONSTRAINT "fk_resource_permissions_role" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE TABLE IF NOT EXISTS user_role_assignments(
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "role_id" TEXT NOT NULL,
    "expires_at" INTEGER,
    "assigned_by" TEXT,
    "assigned_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_user_role_assignments" PRIMARY KEY ("id"),
    CONSTRAINT "fk_user_role_assignments_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_role_assignments_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_role_assignments_role" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_role_assignments_assigned_by" FOREIGN KEY ("assigned_by") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL
);

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_role_assignments_user_org ON user_role_assignments ("user_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS idx_user_role_assignments_role_id ON user_role_assignments ("role_id");
