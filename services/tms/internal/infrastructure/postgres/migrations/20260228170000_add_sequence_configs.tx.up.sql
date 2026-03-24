--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--
CREATE TABLE IF NOT EXISTS "sequence_configs"(
    "id" varchar(100) PRIMARY KEY,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "sequence_type" sequence_type_enum NOT NULL,
    "prefix" varchar(20) NOT NULL,
    "include_year" boolean NOT NULL DEFAULT TRUE,
    "year_digits" int2 NOT NULL DEFAULT 2 CHECK ("year_digits" BETWEEN 2 AND 4),
    "include_month" boolean NOT NULL DEFAULT TRUE,
    "include_week_number" boolean NOT NULL DEFAULT FALSE,
    "include_day" boolean NOT NULL DEFAULT FALSE,
    "sequence_digits" int2 NOT NULL CHECK ("sequence_digits" BETWEEN 1 AND 10),
    "include_location_code" boolean NOT NULL DEFAULT FALSE,
    "location_code" varchar(20) NOT NULL DEFAULT '',
    "include_random_digits" boolean NOT NULL DEFAULT FALSE,
    "random_digits_count" int2 NOT NULL DEFAULT 0 CHECK ("random_digits_count" BETWEEN 0 AND 10),
    "include_check_digit" boolean NOT NULL DEFAULT FALSE,
    "include_business_unit_code" boolean NOT NULL DEFAULT FALSE,
    "business_unit_code" varchar(20) NOT NULL DEFAULT '',
    "use_separators" boolean NOT NULL DEFAULT FALSE,
    "separator_char" varchar(2) NOT NULL DEFAULT '',
    "allow_custom_format" boolean NOT NULL DEFAULT FALSE,
    "custom_format" text NOT NULL DEFAULT '',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp) ::bigint,
    CONSTRAINT "fk_sequence_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON DELETE CASCADE,
    CONSTRAINT "fk_sequence_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON DELETE CASCADE,
    CONSTRAINT "uk_sequence_configs_unique_key" UNIQUE ("sequence_type", "organization_id", "business_unit_id")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_sequence_configs_org_type" ON "sequence_configs"("organization_id", "sequence_type");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_sequence_configs_org_bu_type" ON "sequence_configs"("organization_id", "business_unit_id", "sequence_type");

