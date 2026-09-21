-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231002100_watchtower.tx.up.sql

CREATE TABLE IF NOT EXISTS "watchtower_items"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "source_kind" TEXT NOT NULL,
    "source_id" TEXT NOT NULL,
    "severity" TEXT NOT NULL,
    "title" TEXT NOT NULL,
    "summary" TEXT,
    "subject_type" TEXT,
    "subject_id" TEXT,
    "event_kind" TEXT,
    "path" TEXT,
    "occurred_at" INTEGER NOT NULL,
    "resolved_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_watchtower_items" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_watchtower_items_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_items_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_watchtower_items_severity" CHECK ("severity" IN ('Info', 'Warning', 'Critical'))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_watchtower_items_source" ON "watchtower_items" ("organization_id", "business_unit_id", "source_kind", "source_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_watchtower_items_open" ON "watchtower_items" ("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC)WHERE "resolved_at" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_watchtower_items_feed" ON "watchtower_items" ("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_watchtower_items_resolved" ON "watchtower_items" ("resolved_at")WHERE "resolved_at" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "watchtower_cursors"(
    "user_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "seen_at" INTEGER NOT NULL DEFAULT 0,
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_watchtower_cursors" PRIMARY KEY ("user_id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_watchtower_cursors_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_cursors_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_watchtower_cursors_user" FOREIGN KEY ("user_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
