-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250616062950_notification_preferences.tx.up.sql

CREATE TABLE IF NOT EXISTS "notification_preferences"(
    "id" TEXT PRIMARY KEY,
    "user_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "resource" TEXT NOT NULL,
    "update_types" TEXT NOT NULL DEFAULT '[]',
    "notify_on_all_updates" INTEGER NOT NULL DEFAULT 0,
    "notify_only_owned_records" INTEGER NOT NULL DEFAULT 1,
    "excluded_user_ids" TEXT,
    "included_role_ids" TEXT,
    "preferred_channels" TEXT NOT NULL DEFAULT '{user}',
    "quiet_hours_enabled" INTEGER NOT NULL DEFAULT 0,
    "quiet_hours_start" TEXT,
    "quiet_hours_end" TEXT,
    "timezone" TEXT NOT NULL DEFAULT 'UTC',
    "batch_notifications" INTEGER NOT NULL DEFAULT 0,
    "batch_interval_minutes" INTEGER NOT NULL DEFAULT 15,
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "fk_notification_preferences_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notification_preferences_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notification_preferences_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_batch_interval" CHECK ("batch_interval_minutes" >= 1 AND "batch_interval_minutes" <= 1440)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notification_preferences_user" ON "notification_preferences" ("user_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notification_preferences_org" ON "notification_preferences" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notification_preferences_resource" ON "notification_preferences" ("resource");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notification_preferences_active" ON "notification_preferences" ("is_active")WHERE
    "is_active" = TRUE;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notification_preferences_user_resource_active" ON "notification_preferences" ("user_id", "resource", "is_active")WHERE
    "is_active" = TRUE;
