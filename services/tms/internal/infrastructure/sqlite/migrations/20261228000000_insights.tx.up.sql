-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261228000000_insights.tx.up.sql

CREATE TABLE IF NOT EXISTS "insights"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "detector_key" TEXT NOT NULL,
    "category" TEXT NOT NULL,
    "severity" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "dedupe_key" TEXT NOT NULL,
    "subject" TEXT,
    "headline" TEXT NOT NULL,
    "narrative" TEXT,
    "recommendation" TEXT,
    "narrated" INTEGER NOT NULL DEFAULT 0,
    "metrics" TEXT NOT NULL DEFAULT '[]',
    "links" TEXT NOT NULL DEFAULT '[]',
    "window_start" INTEGER NOT NULL DEFAULT 0,
    "window_end" INTEGER NOT NULL,
    "detected_at" INTEGER NOT NULL,
    "stale_at" INTEGER NOT NULL,
    "dismissed_at" INTEGER,
    "dismissed_by_id" TEXT,
    "dismiss_reason" TEXT,
    "model_identifier" TEXT,
    "provider_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_insights" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_insights_org" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_insights_bu" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_insights_dismissed_by" FOREIGN KEY ("dismissed_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_insights_category" CHECK ("category" IN ('ServiceQuality', 'CashFlow', 'CostLeakage', 'Compliance')),
    CONSTRAINT "ck_insights_severity" CHECK ("severity" IN ('Info', 'Warning', 'Critical')),
    CONSTRAINT "ck_insights_status" CHECK ("status" IN ('Active', 'Dismissed', 'Resolved', 'Superseded')),
    CONSTRAINT "ck_insights_window" CHECK ("window_start" <= "window_end"),
    CONSTRAINT "ck_insights_dismissed_at" CHECK ("status" <> 'Dismissed' OR "dismissed_at" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_insights_active_dedupe" ON "insights" ("organization_id", "business_unit_id", "dedupe_key")WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_insights_active" ON "insights" ("organization_id", "business_unit_id", "status", "detected_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_insights_dismissed_dedupe" ON "insights" ("organization_id", "business_unit_id", "dedupe_key", "dismissed_at" DESC)WHERE
    "status" = 'Dismissed';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_insights_detector" ON "insights" ("organization_id", "business_unit_id", "detector_key", "status");
