CREATE TYPE "rate_geo_scope_type_enum" AS ENUM(
    'Any',
    'Country',
    'State',
    'Zone',
    'Radius',
    'CityState',
    'Zip3',
    'Zip5',
    'Location'
);

--bun:split
CREATE TYPE "rate_rounding_mode_enum" AS ENUM(
    'HalfUp',
    'HalfEven',
    'Up',
    'Down',
    'None'
);

--bun:split
CREATE TYPE "rate_zone_kind_enum" AS ENUM(
    'Custom',
    'KMA',
    'Regional',
    'Metro',
    'Country'
);

--bun:split
CREATE TYPE "rate_matrix_value_kind_enum" AS ENUM(
    'FlatRate',
    'PerMile',
    'PerCwt',
    'PerPiece',
    'PerStop',
    'Percent',
    'Discount',
    'MinimumOnly'
);

--bun:split
CREATE TYPE "rate_matrix_dimension_kind_enum" AS ENUM(
    'Zone',
    'Zip3',
    'Zip5',
    'State',
    'Country',
    'WeightBreak',
    'Distance',
    'PieceCount',
    'LinearFeet',
    'FreightClass',
    'EquipmentType',
    'ServiceType',
    'Custom'
);

--bun:split
CREATE TYPE "rate_matrix_match_mode_enum" AS ENUM(
    'Exact',
    'Range'
);

--bun:split
CREATE TABLE IF NOT EXISTS "rate_zones"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "kind" rate_zone_kind_enum NOT NULL DEFAULT 'Custom',
    "status" status_enum NOT NULL DEFAULT 'Active',
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_zones" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_zones_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_zones_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_zones_code" ON "rate_zones"("organization_id", "business_unit_id", lower("code"));

--bun:split
ALTER TABLE "rate_zones"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_zones_search" ON "rate_zones" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE rate_zones IS 'Named market areas a contract can be priced against; one zone stands in for hundreds of postal prefixes';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_zone_members"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_zone_id" varchar(100) NOT NULL,
    "scope_type" rate_geo_scope_type_enum NOT NULL,
    "scope_value" varchar(120),
    "city" varchar(100),
    "match_key" varchar(160) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_zone_members" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_zone_members_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_zone_members_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_zone_members_zone" FOREIGN KEY ("rate_zone_id", "organization_id", "business_unit_id") REFERENCES "rate_zones"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Zones are unions of primitive places. Nesting would turn membership from
    -- an indexed lookup into a graph walk, and rating cannot afford that.
    CONSTRAINT "chk_rate_zone_members_no_nesting" CHECK ("scope_type" NOT IN ('Zone', 'Any', 'Radius')),
    CONSTRAINT "chk_rate_zone_members_city" CHECK ("scope_type" <> 'CityState' OR "city" IS NOT NULL)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_zone_members" ON "rate_zone_members"("organization_id", "business_unit_id", "rate_zone_id", "match_key");

--bun:split
-- The lookup that turns a shipment's place into the zones it belongs to. It
-- runs twice per rating, so it is served entirely from this index.
CREATE INDEX IF NOT EXISTS "idx_rate_zone_members_match" ON "rate_zone_members"("organization_id", "business_unit_id", "match_key") INCLUDE ("rate_zone_id");

--bun:split
COMMENT ON TABLE rate_zone_members IS 'The places that make up a zone, named with the same scope vocabulary a rate rule uses';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_matrices"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(64) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "value_kind" rate_matrix_value_kind_enum NOT NULL,
    "currency" varchar(3) NOT NULL DEFAULT 'USD',
    "status" status_enum NOT NULL DEFAULT 'Active',
    "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    "rounding_precision" smallint NOT NULL DEFAULT 2,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_matrices" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_matrices_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrices_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_matrices_precision" CHECK ("rounding_precision" BETWEEN 0 AND 6)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrices_code" ON "rate_matrices"("organization_id", "business_unit_id", lower("code"));

--bun:split
ALTER TABLE "rate_matrices"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_matrices_search" ON "rate_matrices" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE rate_matrices IS 'Tariffs that vary along up to four axes, looked up one cell at a time rather than loaded whole';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_matrix_dimensions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_matrix_id" varchar(100) NOT NULL,
    "position" smallint NOT NULL,
    "kind" rate_matrix_dimension_kind_enum NOT NULL,
    "match_mode" rate_matrix_match_mode_enum NOT NULL,
    "label" varchar(100),
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_matrix_dimensions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_matrix_dimensions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_dimensions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_matrix_dimensions_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id") REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Position binds a dimension to a fixed pair of columns on every cell, so
    -- it has to stay inside the four the cell provides.
    CONSTRAINT "chk_rate_matrix_dimensions_position" CHECK ("position" BETWEEN 0 AND 3)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_dimensions_position" ON "rate_matrix_dimensions"("organization_id", "business_unit_id", "rate_matrix_id", "position");

--bun:split
CREATE TABLE IF NOT EXISTS "rate_matrix_cells"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_matrix_id" varchar(100) NOT NULL,
    "d0_key" varchar(120) NOT NULL DEFAULT '',
    "d1_key" varchar(120) NOT NULL DEFAULT '',
    "d2_key" varchar(120) NOT NULL DEFAULT '',
    "d3_key" varchar(120) NOT NULL DEFAULT '',
    "d0_min" numeric(19, 4),
    "d0_max" numeric(19, 4),
    "d1_min" numeric(19, 4),
    "d1_max" numeric(19, 4),
    "d2_min" numeric(19, 4),
    "d2_max" numeric(19, 4),
    "d3_min" numeric(19, 4),
    "d3_max" numeric(19, 4),
    "value" numeric(19, 6) NOT NULL,
    "min_charge" numeric(19, 4),
    "deficit_eligible" boolean NOT NULL DEFAULT TRUE,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
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
-- A wholly exact matrix has one cell per key combination, and duplicates would
-- make the rate depend on which row came back first.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_matrix_cells_exact" ON "rate_matrix_cells"("organization_id", "business_unit_id", "rate_matrix_id", "d0_key", "d1_key", "d2_key", "d3_key")
WHERE
    "d0_min" IS NULL AND "d1_min" IS NULL AND "d2_min" IS NULL AND "d3_min" IS NULL;

--bun:split
-- The lookup path. Exact axes lead so the index stays selective, and the banded
-- axes are narrowed by their bounds before the winning band is chosen in Go.
CREATE INDEX IF NOT EXISTS "idx_rate_matrix_cells_lookup" ON "rate_matrix_cells"("organization_id", "business_unit_id", "rate_matrix_id", "d0_key", "d1_key", "d2_key", "d3_key", "d0_min", "d1_min", "d2_min", "d3_min");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_matrix_cells_dates_brin" ON "rate_matrix_cells" USING BRIN(created_at, updated_at) WITH (pages_per_range = 128);

--bun:split
COMMENT ON TABLE rate_matrix_cells IS 'One priced intersection of a matrix axes; key columns hold exact axes and bound columns hold banded ones, by dimension position';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_density_scales"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "code" varchar(64) NOT NULL,
    "name" varchar(100) NOT NULL,
    "description" text,
    "status" status_enum NOT NULL DEFAULT 'Active',
    "effective_from" bigint NOT NULL,
    "is_org_default" boolean NOT NULL DEFAULT FALSE,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_density_scales" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_density_scales_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scales_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scales_code" ON "rate_density_scales"("organization_id", "business_unit_id", lower("code"));

--bun:split
-- Only one scale can be the organization's default, or a rule that does not
-- name one would classify differently depending on which row was read.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scales_default" ON "rate_density_scales"("organization_id", "business_unit_id")
WHERE
    "is_org_default";

--bun:split
COMMENT ON TABLE rate_density_scales IS 'Density to freight class scales; seeded with the 2025 NMFC thirteen tier scale and editable per organization';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_density_scale_tiers"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_density_scale_id" varchar(100) NOT NULL,
    "from_pcf" numeric(10, 4) NOT NULL,
    "to_pcf" numeric(10, 4),
    "freight_class" freight_class_enum NOT NULL,
    "sort_order" smallint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_density_scale_tiers" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_density_scale_tiers_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scale_tiers_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_density_scale_tiers_scale" FOREIGN KEY ("rate_density_scale_id", "organization_id", "business_unit_id") REFERENCES "rate_density_scales"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_density_scale_tiers_from" CHECK ("from_pcf" >= 0),
    CONSTRAINT "chk_rate_density_scale_tiers_band" CHECK ("to_pcf" IS NULL OR "to_pcf" > "from_pcf")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_density_scale_tiers_from" ON "rate_density_scale_tiers"("organization_id", "business_unit_id", "rate_density_scale_id", "from_pcf");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_density_scale_tiers_scale" ON "rate_density_scale_tiers"("organization_id", "business_unit_id", "rate_density_scale_id", "from_pcf");
