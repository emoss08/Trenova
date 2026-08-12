-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250808151003_add_user_preferences.tx.up.sql

CREATE TABLE IF NOT EXISTS "user_preferences"(
    "id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "preferences" TEXT NOT NULL DEFAULT '{"dismissedNotices": [], "dismissedDialogs": [], "uiSettings": {}}',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_user_preferences" PRIMARY KEY ("id"),
    CONSTRAINT "uq_user_preferences_user" UNIQUE ("user_id"),
    CONSTRAINT "fk_user_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_user_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_user_preferences_user" ON "user_preferences" ("user_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_user_preferences_organization" ON "user_preferences" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_user_preferences_created_at" ON "user_preferences" ("created_at", "updated_at");
