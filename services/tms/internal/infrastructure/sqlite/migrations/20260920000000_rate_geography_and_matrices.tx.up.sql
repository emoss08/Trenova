-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260920000000_rate_geography_and_matrices.tx.up.sql

CREATE TABLE IF NOT EXISTS "rate_zones"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "kind" TEXT NOT NULL DEFAULT 'Custom',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_zones" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_zones_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_zones_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_zones_code" ON "rate_zones" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE TABLE IF NOT EXISTS "rate_zone_members"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_zone_id" TEXT NOT NULL,
    "scope_type" TEXT NOT NULL,
    "scope_value" TEXT,
    "city" TEXT,
    "match_key" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_zone_members" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_zone_members_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_zone_members_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_zone_members_zone" FOREIGN KEY ("rate_zone_id", "organization_id", "business_unit_id") REFERENCES "rate_zones"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_zone_members_no_nesting" CHECK ("scope_type" NOT IN ('Zone', 'Any', 'Radius')),
    CONSTRAINT "chk_rate_zone_members_city" CHECK ("scope_type" <> 'CityState' OR "city" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_zone_members" ON "rate_zone_members" ("organization_id", "business_unit_id", "rate_zone_id", "match_key");

--bun:split

CREATE TABLE IF NOT EXISTS "rate_matrices"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "value_kind" TEXT NOT NULL,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "status" TEXT NOT NULL DEFAULT 'Active',
    "rounding_mode" TEXT NOT NULL DEFAULT 'HalfUp',
    "rounding_precision" INTEGER NOT NULL DEFAULT 2,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_matrices" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_matrices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_matrices_precision" CHECK ("rounding_precision" BETWEEN 0 AND 6)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrices_code" ON "rate_matrices" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE TABLE IF NOT EXISTS "rate_matrix_dimensions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_matrix_id" TEXT NOT NULL,
    "position" INTEGER NOT NULL,
    "kind" TEXT NOT NULL,
    "match_mode" TEXT NOT NULL,
    "label" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_matrix_dimensions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_matrix_dimensions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_dimensions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_dimensions_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id") REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_matrix_dimensions_position" CHECK ("position" BETWEEN 0 AND 3)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_dimensions_position" ON "rate_matrix_dimensions" ("organization_id", "business_unit_id", "rate_matrix_id", "position");

--bun:split

CREATE TABLE IF NOT EXISTS "rate_matrix_cells"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_matrix_id" TEXT NOT NULL,
    "d0_key" TEXT NOT NULL DEFAULT '',
    "d1_key" TEXT NOT NULL DEFAULT '',
    "d2_key" TEXT NOT NULL DEFAULT '',
    "d3_key" TEXT NOT NULL DEFAULT '',
    "d0_min" REAL,
    "d0_max" REAL,
    "d1_min" REAL,
    "d1_max" REAL,
    "d2_min" REAL,
    "d2_max" REAL,
    "d3_min" REAL,
    "d3_max" REAL,
    "value" REAL NOT NULL,
    "min_charge" REAL,
    "deficit_eligible" INTEGER NOT NULL DEFAULT 1,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_matrix_cells" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_matrix_cells_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_cells_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_cells_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id") REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_matrix_cells_value" CHECK ("value" >= 0),
    CONSTRAINT "chk_rate_matrix_cells_min_charge" CHECK ("min_charge" IS NULL OR "min_charge" >= 0),
    CONSTRAINT "chk_rate_matrix_cells_d0_band" CHECK ("d0_max" IS NULL OR "d0_min" IS NULL OR "d0_max" > "d0_min"),
    CONSTRAINT "chk_rate_matrix_cells_d1_band" CHECK ("d1_max" IS NULL OR "d1_min" IS NULL OR "d1_max" > "d1_min"),
    CONSTRAINT "chk_rate_matrix_cells_d2_band" CHECK ("d2_max" IS NULL OR "d2_min" IS NULL OR "d2_max" > "d2_min"),
    CONSTRAINT "chk_rate_matrix_cells_d3_band" CHECK ("d3_max" IS NULL OR "d3_min" IS NULL OR "d3_max" > "d3_min")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_cells_exact" ON "rate_matrix_cells" ("organization_id", "business_unit_id", "rate_matrix_id", "d0_key", "d1_key", "d2_key", "d3_key")WHERE
    "d0_min" IS NULL AND "d1_min" IS NULL AND "d2_min" IS NULL AND "d3_min" IS NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_matrix_cells_lookup" ON "rate_matrix_cells" ("organization_id", "business_unit_id", "rate_matrix_id", "d0_key", "d1_key", "d2_key", "d3_key", "d0_min", "d1_min", "d2_min", "d3_min");

--bun:split

CREATE TABLE IF NOT EXISTS "rate_density_scales"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "effective_from" INTEGER NOT NULL,
    "is_org_default" INTEGER NOT NULL DEFAULT 0,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_density_scales" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_density_scales_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scales_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scales_code" ON "rate_density_scales" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scales_default" ON "rate_density_scales" ("organization_id", "business_unit_id")WHERE
    "is_org_default";

--bun:split

CREATE TABLE IF NOT EXISTS "rate_density_scale_tiers"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_density_scale_id" TEXT NOT NULL,
    "from_pcf" REAL NOT NULL,
    "to_pcf" REAL,
    "freight_class" TEXT NOT NULL,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_density_scale_tiers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_density_scale_tiers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scale_tiers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scale_tiers_scale" FOREIGN KEY ("rate_density_scale_id", "organization_id", "business_unit_id") REFERENCES "rate_density_scales"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_density_scale_tiers_from" CHECK ("from_pcf" >= 0),
    CONSTRAINT "chk_rate_density_scale_tiers_band" CHECK ("to_pcf" IS NULL OR "to_pcf" > "from_pcf")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scale_tiers_from" ON "rate_density_scale_tiers" ("organization_id", "business_unit_id", "rate_density_scale_id", "from_pcf");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_density_scale_tiers_scale" ON "rate_density_scale_tiers" ("organization_id", "business_unit_id", "rate_density_scale_id", "from_pcf");
