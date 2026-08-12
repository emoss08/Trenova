-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250608193352_dedicated_lane_config.tx.up.sql

CREATE TABLE IF NOT EXISTS "pattern_configs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "min_frequency" INTEGER NOT NULL DEFAULT 3,
    "analysis_window_days" INTEGER NOT NULL DEFAULT 90,
    "min_confidence_score" NUMERIC NOT NULL DEFAULT 0.7,
    "suggestion_ttl_days" INTEGER NOT NULL DEFAULT 30,
    "require_exact_match" INTEGER NOT NULL DEFAULT 0,
    "weight_recent_shipments" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_pattern_configs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_pattern_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_pattern_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_pattern_configs_min_frequency" CHECK ("min_frequency" >= 1 AND "min_frequency" <= 100),
    CONSTRAINT "chk_pattern_configs_analysis_window_days" CHECK ("analysis_window_days" >= 1 AND "analysis_window_days" <= 365),
    CONSTRAINT "chk_pattern_configs_min_confidence_score" CHECK ("min_confidence_score" >= 0.0 AND "min_confidence_score" <= 1.0),
    CONSTRAINT "chk_pattern_configs_suggestion_ttl_days" CHECK ("suggestion_ttl_days" >= 1 AND "suggestion_ttl_days" <= 365)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_pattern_configs_organization" ON "pattern_configs" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pattern_configs_business_unit" ON "pattern_configs" ("business_unit_id", "organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pattern_configs_created_at" ON "pattern_configs" ("created_at", "updated_at");
