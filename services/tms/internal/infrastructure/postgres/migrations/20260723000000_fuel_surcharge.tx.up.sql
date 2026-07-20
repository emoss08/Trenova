CREATE TYPE "fuel_index_source_enum" AS ENUM('EIA', 'Custom');

--bun:split
CREATE TYPE "fuel_surcharge_method_kind_enum" AS ENUM('PerMileStep', 'PerMileMPG', 'TablePerMile', 'TablePercent', 'TableFlat');

--bun:split
CREATE TYPE "fuel_surcharge_date_basis_enum" AS ENUM('PickupDate', 'TenderDate');

--bun:split
CREATE TYPE "fuel_surcharge_step_rounding_enum" AS ENUM('Up', 'Down', 'Nearest');

--bun:split
CREATE TYPE "fuel_surcharge_rate_rounding_enum" AS ENUM('HalfUp', 'Up', 'Down');

--bun:split
CREATE TYPE "fuel_surcharge_fallback_enum" AS ENUM('UseLatestAvailable', 'Skip');

--bun:split
CREATE TYPE "fuel_surcharge_program_status_enum" AS ENUM('Active', 'Inactive');

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_indices"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "description" text,
    "source" fuel_index_source_enum NOT NULL,
    "eia_series_id" varchar(64),
    "currency" varchar(3) NOT NULL DEFAULT 'USD',
    "is_active" boolean NOT NULL DEFAULT TRUE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce("name", '')), 'A') ||
        setweight(to_tsvector('english', coalesce("code", '')), 'A') ||
        setweight(to_tsvector('english', coalesce("description", '')), 'B')
    ) STORED,
    CONSTRAINT "pk_fuel_indices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_fuel_indices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_indices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_fuel_indices_eia_series" CHECK (("source" = 'EIA' AND "eia_series_id" IS NOT NULL) OR ("source" = 'Custom' AND "eia_series_id" IS NULL)),
    CONSTRAINT "chk_fuel_indices_currency_upper" CHECK (currency = upper(currency))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_indices_code" ON "fuel_indices"("organization_id", "business_unit_id", "code");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_indices_bu_org" ON "fuel_indices"("business_unit_id", "organization_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_indices_search" ON "fuel_indices" USING gin("search_vector");

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_index_prices"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "fuel_index_id" varchar(100) NOT NULL,
    "price_date" date NOT NULL,
    "price" numeric(19, 4) NOT NULL,
    "currency" varchar(3) NOT NULL DEFAULT 'USD',
    "is_manual" boolean NOT NULL DEFAULT FALSE,
    "entered_by_id" varchar(100),
    "source_raw" varchar(64),
    "fetched_at" timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "pk_fuel_index_prices" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_fuel_index_prices_fuel_index" FOREIGN KEY ("fuel_index_id", "business_unit_id", "organization_id") REFERENCES "fuel_indices"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_index_prices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_index_prices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "uq_fuel_index_prices_week" UNIQUE ("organization_id", "business_unit_id", "fuel_index_id", "price_date"),
    CONSTRAINT "chk_fuel_index_prices_price_positive" CHECK (price > 0)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_index_prices_lookup" ON "fuel_index_prices"("fuel_index_id", "organization_id", "business_unit_id", "price_date" DESC);

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_surcharge_programs"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "description" text,
    "status" fuel_surcharge_program_status_enum NOT NULL DEFAULT 'Active',
    "fuel_index_id" varchar(100) NOT NULL,
    "accessorial_charge_id" varchar(100) NOT NULL,
    "method" fuel_surcharge_method_kind_enum NOT NULL,
    "peg_price" numeric(19, 4),
    "increment" numeric(19, 4),
    "increment_rate" numeric(19, 4),
    "miles_per_gallon" numeric(9, 2),
    "step_rounding" fuel_surcharge_step_rounding_enum NOT NULL DEFAULT 'Up',
    "rate_rounding" fuel_surcharge_rate_rounding_enum NOT NULL DEFAULT 'HalfUp',
    "rate_precision" smallint NOT NULL DEFAULT 4,
    "min_amount" numeric(19, 4),
    "max_amount" numeric(19, 4),
    "date_basis" fuel_surcharge_date_basis_enum NOT NULL DEFAULT 'PickupDate',
    "price_effective_day" smallint NOT NULL DEFAULT 3,
    "missing_price_fallback" fuel_surcharge_fallback_enum NOT NULL DEFAULT 'UseLatestAvailable',
    "effective_start_date" bigint,
    "effective_end_date" bigint,
    "shipment_type_ids" jsonb,
    "service_type_ids" jsonb,
    "tractor_type_ids" jsonb,
    "trailer_type_ids" jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "search_vector" tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce("name", '')), 'A') ||
        setweight(to_tsvector('english', coalesce("code", '')), 'A') ||
        setweight(to_tsvector('english', coalesce("description", '')), 'B')
    ) STORED,
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
CREATE UNIQUE INDEX IF NOT EXISTS "uq_fuel_surcharge_programs_code" ON "fuel_surcharge_programs"("organization_id", "business_unit_id", "code");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_surcharge_programs_status" ON "fuel_surcharge_programs"("organization_id", "business_unit_id", "status");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_surcharge_programs_search" ON "fuel_surcharge_programs" USING gin("search_vector");

--bun:split
CREATE TABLE IF NOT EXISTS "fuel_surcharge_table_rows"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "fuel_surcharge_program_id" varchar(100) NOT NULL,
    "price_min" numeric(19, 4),
    "price_max" numeric(19, 4),
    "value" numeric(19, 4) NOT NULL,
    "sort_order" integer NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_fuel_surcharge_table_rows" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_fuel_surcharge_table_rows_program" FOREIGN KEY ("fuel_surcharge_program_id", "business_unit_id", "organization_id") REFERENCES "fuel_surcharge_programs"("id", "business_unit_id", "organization_id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_surcharge_table_rows_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_fuel_surcharge_table_rows_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "chk_fuel_surcharge_table_rows_band" CHECK (price_min IS NULL OR price_max IS NULL OR price_max > price_min)
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_fuel_surcharge_table_rows_program" ON "fuel_surcharge_table_rows"("fuel_surcharge_program_id", "organization_id", "business_unit_id", "sort_order");

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD COLUMN IF NOT EXISTS "fuel_surcharge_program_id" varchar(100);

--bun:split
ALTER TABLE "customer_billing_profiles"
    ADD CONSTRAINT "fk_customer_billing_profiles_fuel_surcharge_program" FOREIGN KEY ("fuel_surcharge_program_id", "business_unit_id", "organization_id") REFERENCES "fuel_surcharge_programs"("id", "business_unit_id", "organization_id") ON DELETE SET NULL;

--bun:split
ALTER TABLE "additional_charges"
    ADD COLUMN IF NOT EXISTS "fuel_surcharge_program_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "fuel_surcharge_detail" jsonb;
