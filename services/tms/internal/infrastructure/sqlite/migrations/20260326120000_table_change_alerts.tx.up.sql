-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260326120000_table_change_alerts.tx.up.sql

CREATE TABLE IF NOT EXISTS "tca_allowlisted_tables" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "table_name" TEXT NOT NULL,
    "display_name" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_tca_allowlist_org" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "fk_tca_allowlist_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id"),
    UNIQUE ("organization_id", "business_unit_id", "table_name")
);

--bun:split

CREATE TABLE IF NOT EXISTS "tca_subscriptions" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "table_name" TEXT NOT NULL,
    "record_id" TEXT,
    "event_types" TEXT NOT NULL DEFAULT '["INSERT","UPDATE","DELETE"]',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_tca_sub_org" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "fk_tca_sub_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id"),
    CONSTRAINT "fk_tca_sub_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tca_subscriptions_lookup"
    ON "tca_subscriptions" ("organization_id", "business_unit_id", "table_name", "status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_tca_subscriptions_user"
    ON "tca_subscriptions" ("organization_id", "business_unit_id", "user_id");
