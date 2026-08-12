-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250616010742_notifications.tx.up.sql

CREATE TABLE IF NOT EXISTS "notifications"(
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT,
    "target_user_id" TEXT,
    "event_type" TEXT NOT NULL,
    "priority" TEXT NOT NULL DEFAULT 'medium',
    "channel" TEXT NOT NULL DEFAULT 'global',
    "title" TEXT NOT NULL,
    "message" TEXT NOT NULL,
    "data" TEXT,
    "related_entities" TEXT,
    "actions" TEXT,
    "expires_at" INTEGER,
    "delivered_at" INTEGER,
    "read_at" INTEGER,
    "dismissed_at" INTEGER,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "delivery_status" TEXT NOT NULL DEFAULT 'pending',
    "retry_count" INTEGER NOT NULL DEFAULT 0,
    "max_retries" INTEGER NOT NULL DEFAULT 3,
    "source" TEXT NOT NULL,
    "job_id" TEXT,
    "correlation_id" TEXT,
    "tags" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY ("id"),
    CONSTRAINT "fk_notifications_organization_id" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notifications_business_unit_id" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_notifications_target_user_id" FOREIGN KEY ("target_user_id") REFERENCES "users"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_notifications_priority" CHECK ("priority" IN ('critical', 'high', 'medium', 'low')),
    CONSTRAINT "chk_notifications_channel" CHECK ("channel" IN ('global', 'user', 'role')),
    CONSTRAINT "chk_notifications_delivery_status" CHECK ("delivery_status" IN ('pending', 'delivered', 'failed', 'expired')),
    CONSTRAINT "chk_notifications_retry_count" CHECK ("retry_count" >= 0 AND "retry_count" <= "max_retries"),
    CONSTRAINT "chk_notifications_max_retries" CHECK ("max_retries" >= 0 AND "max_retries" <= 10),
    CONSTRAINT "chk_notifications_user_channel" CHECK (("channel" = 'user' AND "target_user_id" IS NOT NULL) OR ("channel" != 'user'))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notifications_organization" ON "notifications" ("organization_id", "created_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notifications_user" ON "notifications" ("target_user_id", "organization_id", "read_at", "created_at" DESC)WHERE
    "target_user_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notifications_unread" ON "notifications" ("organization_id", "read_at", "expires_at", "created_at" DESC)WHERE
    "read_at" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notifications_delivery" ON "notifications" ("delivery_status", "retry_count", "max_retries", "expires_at")WHERE
    "delivery_status" IN ('failed', 'pending');

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notifications_cleanup" ON "notifications" ("created_at", "read_at", "dismissed_at", "expires_at");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notifications_job" ON "notifications" ("job_id", "source", "created_at" DESC)WHERE
    "job_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_notifications_event_type" ON "notifications" ("event_type", "organization_id", "created_at" DESC);
