-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260906000000_jurisdiction_rules.tx.up.sql

CREATE TABLE IF NOT EXISTS "jurisdiction_rules"(
    "id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "max_width_feet" REAL NOT NULL,
    "max_height_feet" REAL NOT NULL,
    "max_length_feet" REAL NOT NULL,
    "max_weight_pounds" INTEGER NOT NULL,
    "superload_width_feet" REAL,
    "superload_weight_pounds" INTEGER,
    "daylight_only" INTEGER NOT NULL DEFAULT 0,
    "rush_hour_restricted" INTEGER NOT NULL DEFAULT 0,
    "weekend_restricted" INTEGER NOT NULL DEFAULT 0,
    "holiday_restricted" INTEGER NOT NULL DEFAULT 1,
    "permit_lead_time_days" INTEGER NOT NULL DEFAULT 1,
    "permit_validity_days" INTEGER NOT NULL DEFAULT 5,
    "permit_base_fee" REAL,
    "permit_per_mile_fee" REAL,
    "source_note" TEXT,
    "source_url" TEXT,
    "verification_state" TEXT NOT NULL DEFAULT 'Unverified',
    "verified_at" INTEGER,
    "effective_start_date" INTEGER,
    "effective_end_date" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_jurisdiction_rules" PRIMARY KEY ("id"),
    CONSTRAINT "fk_jurisdiction_rules_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_jurisdiction_rules_limits" CHECK ("max_width_feet" > 0 AND "max_height_feet" > 0 AND "max_length_feet" > 0 AND "max_weight_pounds" > 0),
    CONSTRAINT "chk_jurisdiction_rules_lead_time" CHECK ("permit_lead_time_days" >= 0 AND "permit_lead_time_days" <= 60),
    CONSTRAINT "chk_jurisdiction_rules_validity" CHECK ("permit_validity_days" > 0 AND "permit_validity_days" <= 365),
    CONSTRAINT "chk_jurisdiction_rules_effective_window" CHECK ("effective_start_date" IS NULL OR "effective_end_date" IS NULL OR "effective_end_date" > "effective_start_date")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_jurisdiction_rules_state" ON "jurisdiction_rules" ("state_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "jurisdiction_escort_thresholds"(
    "id" TEXT NOT NULL,
    "jurisdiction_rule_id" TEXT NOT NULL,
    "role" TEXT NOT NULL,
    "trigger" TEXT NOT NULL,
    "threshold_value" REAL NOT NULL,
    "note" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_jurisdiction_escort_thresholds" PRIMARY KEY ("id"),
    CONSTRAINT "fk_jurisdiction_escort_thresholds_rule" FOREIGN KEY ("jurisdiction_rule_id") REFERENCES "jurisdiction_rules"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_jurisdiction_escort_thresholds_positive" CHECK ("threshold_value" > 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_jurisdiction_escort_thresholds" ON "jurisdiction_escort_thresholds" ("jurisdiction_rule_id", "role", "trigger");

--bun:split

CREATE TABLE IF NOT EXISTS "jurisdiction_rule_overrides"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "state_id" TEXT NOT NULL,
    "max_width_feet" REAL,
    "max_height_feet" REAL,
    "max_length_feet" REAL,
    "max_weight_pounds" INTEGER,
    "permit_lead_time_days" INTEGER,
    "daylight_only" INTEGER,
    "holiday_restricted" INTEGER,
    "reason" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_jurisdiction_rule_overrides" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_jurisdiction_rule_overrides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_jurisdiction_rule_overrides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_jurisdiction_rule_overrides_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_jurisdiction_rule_overrides_reason" CHECK (length("reason") >= 10)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_jurisdiction_rule_overrides_state" ON "jurisdiction_rule_overrides" ("organization_id", "business_unit_id", "state_id");
