-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260526120000_rbac3_hierarchy_constraints.tx.up.sql

CREATE TABLE IF NOT EXISTS role_hierarchy_edges(
    "id" TEXT NOT NULL,
    "senior_role_id" TEXT NOT NULL,
    "junior_role_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "created_by" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_role_hierarchy_edges" PRIMARY KEY ("id"),
    CONSTRAINT "fk_role_hierarchy_edges_senior_role" FOREIGN KEY ("senior_role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_hierarchy_edges_junior_role" FOREIGN KEY ("junior_role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_hierarchy_edges_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_hierarchy_edges_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_role_hierarchy_edges_not_self" CHECK ("senior_role_id" <> "junior_role_id")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uniq_role_hierarchy_edges_senior_junior"
    ON role_hierarchy_edges ("senior_role_id", "junior_role_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_role_hierarchy_edges_junior"
    ON role_hierarchy_edges ("junior_role_id");

--bun:split

CREATE TABLE IF NOT EXISTS role_constraints(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "type" TEXT NOT NULL,
    "max_roles" INTEGER NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_by" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_role_constraints" PRIMARY KEY ("id"),
    CONSTRAINT "fk_role_constraints_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraints_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_role_constraints_type" CHECK ("type" IN ('ssd', 'dsd')),
    CONSTRAINT "chk_role_constraints_max_roles" CHECK ("max_roles" >= 1)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_role_constraints_org_type"
    ON role_constraints ("organization_id", "type");

--bun:split

CREATE TABLE IF NOT EXISTS role_constraint_roles(
    "id" TEXT NOT NULL,
    "role_constraint_id" TEXT NOT NULL,
    "role_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_role_constraint_roles" PRIMARY KEY ("id"),
    CONSTRAINT "fk_role_constraint_roles_constraint" FOREIGN KEY ("role_constraint_id") REFERENCES "role_constraints"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraint_roles_role" FOREIGN KEY ("role_id") REFERENCES "roles"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraint_roles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_role_constraint_roles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uniq_role_constraint_roles_constraint_role"
    ON role_constraint_roles ("role_constraint_id", "role_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_role_constraint_roles_role"
    ON role_constraint_roles ("role_id");
