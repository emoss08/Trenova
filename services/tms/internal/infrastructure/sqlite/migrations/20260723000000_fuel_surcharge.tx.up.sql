-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260723000000_fuel_surcharge.tx.up.sql

CREATE TABLE IF NOT EXISTS "fuel_indices"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "description" TEXT,
    "source" TEXT NOT NULL,
    "eia_series_id" TEXT,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fuel_indices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_fuel_indices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_indices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_fuel_indices_eia_series" CHECK (("source" = 'EIA' AND "eia_series_id" IS NOT NULL) OR ("source" = 'Custom' AND "eia_series_id" IS NULL)),
    CONSTRAINT "chk_fuel_indices_currency_upper" CHECK (currency = upper(currency))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_indices_code" ON "fuel_indices" ("organization_id", "business_unit_id", "code");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_indices_bu_org" ON "fuel_indices" ("business_unit_id", "organization_id");

--bun:split

CREATE TABLE IF NOT EXISTS "fuel_index_prices"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "fuel_index_id" TEXT NOT NULL,
    "price_date" TEXT NOT NULL,
    "price" NUMERIC NOT NULL,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "is_manual" INTEGER NOT NULL DEFAULT 0,
    "entered_by_id" TEXT,
    "source_raw" TEXT,
    "fetched_at" TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_fuel_index_prices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_fuel_index_prices_fuel_index" FOREIGN KEY ("fuel_index_id", "business_unit_id", "organization_id") REFERENCES "fuel_indices"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_index_prices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_index_prices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "uq_fuel_index_prices_week" UNIQUE ("organization_id", "business_unit_id", "fuel_index_id", "price_date"),
    CONSTRAINT "chk_fuel_index_prices_price_positive" CHECK (price > 0)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_index_prices_lookup" ON "fuel_index_prices" ("fuel_index_id", "organization_id", "business_unit_id", "price_date" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "fuel_surcharge_programs"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "fuel_index_id" TEXT NOT NULL,
    "accessorial_charge_id" TEXT NOT NULL,
    "method" TEXT NOT NULL,
    "peg_price" NUMERIC,
    "increment" NUMERIC,
    "increment_rate" NUMERIC,
    "miles_per_gallon" NUMERIC,
    "step_rounding" TEXT NOT NULL DEFAULT 'Up',
    "rate_rounding" TEXT NOT NULL DEFAULT 'HalfUp',
    "rate_precision" INTEGER NOT NULL DEFAULT 4,
    "min_amount" NUMERIC,
    "max_amount" NUMERIC,
    "date_basis" TEXT NOT NULL DEFAULT 'PickupDate',
    "price_effective_day" INTEGER NOT NULL DEFAULT 3,
    "missing_price_fallback" TEXT NOT NULL DEFAULT 'UseLatestAvailable',
    "effective_start_date" INTEGER,
    "effective_end_date" INTEGER,
    "shipment_type_ids" TEXT,
    "service_type_ids" TEXT,
    "tractor_type_ids" TEXT,
    "trailer_type_ids" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fuel_surcharge_programs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_fuel_surcharge_programs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_surcharge_programs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_surcharge_programs_fuel_index" FOREIGN KEY ("fuel_index_id", "business_unit_id", "organization_id") REFERENCES "fuel_indices"("id", "business_unit_id", "organization_id") ON DELETE RESTRICT,
    CONSTRAINT "fk_fuel_surcharge_programs_accessorial_charge" FOREIGN KEY ("accessorial_charge_id", "business_unit_id", "organization_id") REFERENCES "accessorial_charges"("id", "business_unit_id", "organization_id") ON DELETE RESTRICT,
    CONSTRAINT "chk_fuel_surcharge_programs_effective_day" CHECK (price_effective_day BETWEEN 0 AND 6),
    CONSTRAINT "chk_fuel_surcharge_programs_rate_precision" CHECK (rate_precision BETWEEN 0 AND 6),
    CONSTRAINT "chk_fuel_surcharge_programs_amount_bounds" CHECK (min_amount IS NULL OR max_amount IS NULL OR min_amount <= max_amount)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_surcharge_programs_code" ON "fuel_surcharge_programs" ("organization_id", "business_unit_id", "code");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_surcharge_programs_status" ON "fuel_surcharge_programs" ("organization_id", "business_unit_id", "status");

--bun:split

CREATE TABLE IF NOT EXISTS "fuel_surcharge_table_rows"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "fuel_surcharge_program_id" TEXT NOT NULL,
    "price_min" NUMERIC,
    "price_max" NUMERIC,
    "value" NUMERIC NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_fuel_surcharge_table_rows" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_fuel_surcharge_table_rows_program" FOREIGN KEY ("fuel_surcharge_program_id", "business_unit_id", "organization_id") REFERENCES "fuel_surcharge_programs"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_surcharge_table_rows_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_surcharge_table_rows_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_fuel_surcharge_table_rows_band" CHECK (price_min IS NULL OR price_max IS NULL OR price_max > price_min)
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_fuel_surcharge_table_rows_program" ON "fuel_surcharge_table_rows" ("fuel_surcharge_program_id", "organization_id", "business_unit_id", "sort_order");

--bun:split

ALTER TABLE "customer_billing_profiles" ADD COLUMN "fuel_surcharge_program_id" TEXT;

--bun:split

ALTER TABLE "additional_charges" ADD COLUMN "fuel_surcharge_program_id" TEXT;

--bun:split

ALTER TABLE "additional_charges" ADD COLUMN "fuel_surcharge_detail" TEXT;
