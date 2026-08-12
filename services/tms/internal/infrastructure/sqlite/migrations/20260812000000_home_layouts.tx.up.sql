-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260812000000_home_layouts.tx.up.sql

CREATE TABLE IF NOT EXISTS "home_layout_presets"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "layout" TEXT NOT NULL,
    "role_ids" TEXT,
    "core_responsibility" TEXT,
    "is_org_default" INTEGER NOT NULL DEFAULT 0,
    "locked" INTEGER NOT NULL DEFAULT 0,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "created_by" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_home_layout_presets" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_home_layout_presets_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_layout_presets_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_layout_presets_created_by" FOREIGN KEY ("created_by") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "check_home_layout_presets_priority" CHECK ("priority" BETWEEN 0 AND 1000),
    CONSTRAINT "check_home_layout_presets_core_responsibility" CHECK ("core_responsibility" IS NULL OR "core_responsibility" IN ('Billing', 'Operations', 'Finance', 'Leadership'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_home_layout_presets_tenant" ON "home_layout_presets" ("organization_id", "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_home_layout_presets_name" ON "home_layout_presets" (lower("name"), "organization_id", "business_unit_id");

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_home_layout_presets_org_default" ON "home_layout_presets" ("organization_id", "business_unit_id")WHERE "is_org_default";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_home_layout_presets_priority" ON "home_layout_presets" ("organization_id", "business_unit_id", "priority" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "home_preferences"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "preferences" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_home_preferences" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_home_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_home_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "idx_home_preferences_user" ON "home_preferences" ("user_id", "organization_id", "business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_home_preferences_business_unit" ON "home_preferences" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_home_preferences_organization" ON "home_preferences" ("organization_id");
