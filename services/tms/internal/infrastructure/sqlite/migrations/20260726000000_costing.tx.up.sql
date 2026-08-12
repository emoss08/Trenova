-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260726000000_costing.tx.up.sql

CREATE TABLE IF NOT EXISTS "costing_controls"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "fuel_index_id" TEXT,
    "use_live_fuel_price" INTEGER NOT NULL DEFAULT 1,
    "miles_per_gallon" REAL NOT NULL DEFAULT 6.5,
    "include_deadhead_miles" INTEGER NOT NULL DEFAULT 1,
    "gl_actuals_enabled" INTEGER NOT NULL DEFAULT 0,
    "gl_rolling_months" INTEGER NOT NULL DEFAULT 3,
    "planned_monthly_miles" INTEGER,
    "target_margin_percent" REAL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_costing_controls" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_costing_controls_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_costing_controls_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_costing_controls_fuel_index" FOREIGN KEY ("fuel_index_id", "business_unit_id", "organization_id") REFERENCES "fuel_indices"("id", "business_unit_id", "organization_id") ON DELETE SET NULL,
    CONSTRAINT "uq_costing_controls_organization" UNIQUE ("organization_id"),
    CONSTRAINT "chk_costing_controls_mpg" CHECK (miles_per_gallon > 0 AND miles_per_gallon <= 20),
    CONSTRAINT "chk_costing_controls_gl_rolling_months" CHECK (gl_rolling_months BETWEEN 1 AND 12),
    CONSTRAINT "chk_costing_controls_planned_monthly_miles" CHECK (planned_monthly_miles IS NULL OR planned_monthly_miles > 0),
    CONSTRAINT "chk_costing_controls_target_margin" CHECK (target_margin_percent IS NULL OR (target_margin_percent >= 0 AND target_margin_percent <= 100))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_costing_controls_bu_org" ON "costing_controls" ("business_unit_id", "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS "cost_categories"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "costing_control_id" TEXT NOT NULL,
    "category" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "cost_behavior" TEXT NOT NULL,
    "rate_source" TEXT NOT NULL DEFAULT 'Benchmark',
    "benchmark_rate_per_mile" REAL NOT NULL DEFAULT 0,
    "override_rate_per_mile" REAL,
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_cost_categories" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_cost_categories_costing_control" FOREIGN KEY ("costing_control_id", "business_unit_id", "organization_id") REFERENCES "costing_controls"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_categories_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_categories_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_cost_categories_benchmark_rate" CHECK (benchmark_rate_per_mile >= 0),
    CONSTRAINT "chk_cost_categories_override_rate" CHECK (override_rate_per_mile IS NULL OR override_rate_per_mile >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_cost_categories_category" ON "cost_categories" ("organization_id", "costing_control_id", "category")WHERE "category" != 'Custom';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cost_categories_control" ON "cost_categories" ("costing_control_id", "organization_id", "business_unit_id", "sort_order");

--bun:split

CREATE TABLE IF NOT EXISTS "cost_category_gl_accounts"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "cost_category_id" TEXT NOT NULL,
    "gl_account_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_cost_category_gl_accounts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_cost_category_gl_accounts_category" FOREIGN KEY ("cost_category_id", "business_unit_id", "organization_id") REFERENCES "cost_categories"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_category_gl_accounts_gl_account" FOREIGN KEY ("gl_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_category_gl_accounts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_category_gl_accounts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "uq_cost_category_gl_accounts" UNIQUE ("organization_id", "cost_category_id", "gl_account_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_cost_category_gl_accounts_category" ON "cost_category_gl_accounts" ("cost_category_id", "organization_id", "business_unit_id");
