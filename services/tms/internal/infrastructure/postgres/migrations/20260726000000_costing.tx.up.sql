CREATE TYPE "cost_category_type_enum" AS ENUM('DriverWages', 'DriverBenefits', 'Fuel', 'EquipmentPayments', 'Maintenance', 'Insurance', 'Tires', 'Tolls', 'PermitsLicenses', 'Overhead', 'Custom');

--bun:split
CREATE TYPE "cost_behavior_enum" AS ENUM('Fixed', 'Variable');

--bun:split
CREATE TYPE "cost_rate_source_enum" AS ENUM('Benchmark', 'Override', 'GLActual');

--bun:split
CREATE TABLE IF NOT EXISTS "costing_controls"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "fuel_index_id" varchar(100),
    "use_live_fuel_price" boolean NOT NULL DEFAULT TRUE,
    "miles_per_gallon" numeric(6, 2) NOT NULL DEFAULT 6.5,
    "include_deadhead_miles" boolean NOT NULL DEFAULT TRUE,
    "gl_actuals_enabled" boolean NOT NULL DEFAULT FALSE,
    "gl_rolling_months" smallint NOT NULL DEFAULT 3,
    "planned_monthly_miles" bigint,
    "target_margin_percent" numeric(6, 3),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
CREATE INDEX IF NOT EXISTS "idx_costing_controls_bu_org" ON "costing_controls"("business_unit_id", "organization_id");

--bun:split
CREATE TABLE IF NOT EXISTS "cost_categories"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "costing_control_id" varchar(100) NOT NULL,
    "category" cost_category_type_enum NOT NULL,
    "name" varchar(100) NOT NULL,
    "cost_behavior" cost_behavior_enum NOT NULL,
    "rate_source" cost_rate_source_enum NOT NULL DEFAULT 'Benchmark',
    "benchmark_rate_per_mile" numeric(19, 6) NOT NULL DEFAULT 0,
    "override_rate_per_mile" numeric(19, 6),
    "is_active" boolean NOT NULL DEFAULT TRUE,
    "sort_order" smallint NOT NULL DEFAULT 0,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_cost_categories" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_cost_categories_costing_control" FOREIGN KEY ("costing_control_id", "business_unit_id", "organization_id") REFERENCES "costing_controls"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_categories_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_categories_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_cost_categories_benchmark_rate" CHECK (benchmark_rate_per_mile >= 0),
    CONSTRAINT "chk_cost_categories_override_rate" CHECK (override_rate_per_mile IS NULL OR override_rate_per_mile >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_cost_categories_category" ON "cost_categories"("organization_id", "costing_control_id", "category")
WHERE "category" != 'Custom';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_cost_categories_control" ON "cost_categories"("costing_control_id", "organization_id", "business_unit_id", "sort_order");

--bun:split
CREATE TABLE IF NOT EXISTS "cost_category_gl_accounts"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "cost_category_id" varchar(100) NOT NULL,
    "gl_account_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_cost_category_gl_accounts" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_cost_category_gl_accounts_category" FOREIGN KEY ("cost_category_id", "business_unit_id", "organization_id") REFERENCES "cost_categories"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_category_gl_accounts_gl_account" FOREIGN KEY ("gl_account_id", "organization_id", "business_unit_id") REFERENCES "gl_accounts"("id", "organization_id", "business_unit_id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_category_gl_accounts_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_cost_category_gl_accounts_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "uq_cost_category_gl_accounts" UNIQUE ("organization_id", "cost_category_id", "gl_account_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_cost_category_gl_accounts_category" ON "cost_category_gl_accounts"("cost_category_id", "organization_id", "business_unit_id");
