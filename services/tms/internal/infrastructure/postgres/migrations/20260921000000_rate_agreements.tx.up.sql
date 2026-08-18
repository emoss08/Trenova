CREATE TYPE "rate_agreement_party_type_enum" AS ENUM(
    'Customer',
    'Carrier'
);

--bun:split
CREATE TYPE "rate_agreement_type_enum" AS ENUM(
    'Contract',
    'Tariff',
    'Spot',
    'Project',
    'Dedicated'
);

--bun:split
CREATE TYPE "rate_agreement_status_enum" AS ENUM(
    'Draft',
    'InReview',
    'Active',
    'Suspended',
    'Expired',
    'Archived'
);

--bun:split
CREATE TYPE "rate_agreement_rule_status_enum" AS ENUM(
    'Active',
    'Inactive'
);

--bun:split
CREATE TYPE "rate_agreement_basis_enum" AS ENUM(
    'Flat',
    'PerMile',
    'PerCwt',
    'PerPiece',
    'PerStop',
    'PerPallet',
    'PerLinearFoot',
    'PerHour',
    'Percent',
    'Matrix',
    'Formula'
);

--bun:split
CREATE TYPE "rate_agreement_percent_basis_enum" AS ENUM(
    'Linehaul',
    'LinehaulPlusAccessorials',
    'SellRate'
);

--bun:split
CREATE TYPE "rate_agreement_direction_enum" AS ENUM(
    'Directional',
    'Bidirectional'
);

--bun:split
CREATE TYPE "rate_freight_class_source_enum" AS ENUM(
    'Commodity',
    'Density',
    'Fixed'
);

--bun:split
CREATE TYPE "rate_quote_purpose_enum" AS ENUM(
    'Rating',
    'Shopping',
    'Simulation',
    'WhatIf'
);

--bun:split
CREATE TYPE "rate_quote_outcome_enum" AS ENUM(
    'Rated',
    'FormulaFallback',
    'ManualOverride',
    'NoRateFound',
    'Error'
);

--bun:split
CREATE TYPE "rate_quote_status_enum" AS ENUM(
    'Applied',
    'Superseded',
    'Quoted'
);

--bun:split
CREATE TYPE "unrated_shipment_disposition_enum" AS ENUM(
    'FallbackFormulaTemplate',
    'ZeroAndFlag',
    'Block'
);

--bun:split
CREATE TABLE IF NOT EXISTS "rate_agreements"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "party_type" rate_agreement_party_type_enum NOT NULL,
    "customer_id" varchar(100),
    "carrier_id" varchar(100),
    "code" varchar(50) NOT NULL,
    "name" varchar(150) NOT NULL,
    "description" text,
    "agreement_type" rate_agreement_type_enum NOT NULL DEFAULT 'Contract',
    "status" rate_agreement_status_enum NOT NULL DEFAULT 'Draft',
    "contract_ref" varchar(100),
    "document_id" varchar(100),
    "priority" smallint NOT NULL DEFAULT 0,
    "effective_from" bigint NOT NULL,
    "effective_to" bigint,
    "auto_renew" boolean NOT NULL DEFAULT FALSE,
    "renewal_notice_days" smallint NOT NULL DEFAULT 30,
    "currency" varchar(3) NOT NULL DEFAULT 'USD',
    "default_min_charge" numeric(19, 4),
    "default_max_charge" numeric(19, 4),
    "rounding_mode" rate_rounding_mode_enum NOT NULL DEFAULT 'HalfUp',
    "rounding_precision" smallint NOT NULL DEFAULT 2,
    "bill_to_customer_id" varchar(100),
    "margin_floor_percent" numeric(9, 4),
    "max_pay_percent_of_sell" numeric(9, 4),
    "submitted_by_id" varchar(100),
    "submitted_at" bigint,
    "approved_by_id" varchar(100),
    "approved_at" bigint,
    "review_comment" text,
    "current_version_number" bigint NOT NULL DEFAULT 1,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_agreements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreements_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreements_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreements_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreements_bill_to" FOREIGN KEY ("bill_to_customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreements_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    -- The discriminator and the two party keys have to agree. An agreement
    -- naming the wrong side would resolve against neither party and price
    -- nothing, which is far harder to diagnose later than a rejected write.
    CONSTRAINT "chk_rate_agreements_party" CHECK (("party_type" = 'Customer' AND "customer_id" IS NOT NULL AND "carrier_id" IS NULL) OR ("party_type" = 'Carrier' AND "carrier_id" IS NOT NULL AND "customer_id" IS NULL)),
    CONSTRAINT "chk_rate_agreements_sell_side_fields" CHECK ("party_type" = 'Customer' OR "bill_to_customer_id" IS NULL),
    CONSTRAINT "chk_rate_agreements_buy_side_fields" CHECK ("party_type" = 'Carrier' OR ("margin_floor_percent" IS NULL AND "max_pay_percent_of_sell" IS NULL)),
    CONSTRAINT "chk_rate_agreements_window" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from"),
    CONSTRAINT "chk_rate_agreements_charges" CHECK ("default_min_charge" IS NULL OR "default_max_charge" IS NULL OR "default_max_charge" >= "default_min_charge"),
    CONSTRAINT "chk_rate_agreements_precision" CHECK ("rounding_precision" BETWEEN 0 AND 6)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreements_code" ON "rate_agreements"("organization_id", "business_unit_id", lower("code"));

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreements_customer" ON "rate_agreements"("organization_id", "business_unit_id", "customer_id", "effective_from" DESC)
WHERE
    "status" = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreements_carrier" ON "rate_agreements"("organization_id", "business_unit_id", "carrier_id", "effective_from" DESC)
WHERE
    "status" = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreements_expiring" ON "rate_agreements"("organization_id", "business_unit_id", "effective_to")
WHERE
    "status" = 'Active' AND "effective_to" IS NOT NULL;

--bun:split
ALTER TABLE "rate_agreements"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("contract_ref", '')), 'B') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'C')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreements_search" ON "rate_agreements" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE rate_agreements IS 'Commercial contracts a shipment is priced under; the header carries negotiated terms while the rules beneath carry their own effective windows';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_agreement_versions"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_agreement_id" varchar(100) NOT NULL,
    "version_number" bigint NOT NULL,
    "effective_from" bigint NOT NULL,
    "effective_to" bigint,
    "name" varchar(150) NOT NULL,
    "description" text,
    "agreement_type" rate_agreement_type_enum NOT NULL,
    "status" rate_agreement_status_enum NOT NULL,
    "contract_ref" varchar(100),
    "currency" varchar(3) NOT NULL,
    "default_min_charge" numeric(19, 4),
    "default_max_charge" numeric(19, 4),
    "rounding_mode" rate_rounding_mode_enum NOT NULL,
    "rounding_precision" smallint NOT NULL,
    "margin_floor_percent" numeric(9, 4),
    "max_pay_percent_of_sell" numeric(9, 4),
    "change_message" text,
    "change_summary" jsonb,
    "created_by_id" varchar(100) NOT NULL,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_agreement_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_versions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_versions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_versions_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_versions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_agreement_versions_window" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from")
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_versions_number" ON "rate_agreement_versions"("organization_id", "business_unit_id", "rate_agreement_id", "version_number");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_versions_effective" ON "rate_agreement_versions"("organization_id", "business_unit_id", "rate_agreement_id", "effective_from" DESC);

--bun:split
COMMENT ON TABLE rate_agreement_versions IS 'Header terms as they stood; rules are not copied because they carry their own effective windows';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_agreement_rules"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_agreement_id" varchar(100) NOT NULL,
    "party_type" rate_agreement_party_type_enum NOT NULL,
    "party_id" varchar(100) NOT NULL,
    "label" varchar(150),
    "status" rate_agreement_rule_status_enum NOT NULL DEFAULT 'Active',
    "origin_scope_type" rate_geo_scope_type_enum NOT NULL,
    "origin_scope_value" varchar(120),
    "origin_city" varchar(100),
    "destination_scope_type" rate_geo_scope_type_enum NOT NULL,
    "destination_scope_value" varchar(120),
    "destination_city" varchar(100),
    "lane_key" varchar(255) NOT NULL,
    "direction" rate_agreement_direction_enum NOT NULL DEFAULT 'Directional',
    "origin_radius_meters" double precision,
    "destination_radius_meters" double precision,
    "origin_latitude" double precision,
    "origin_longitude" double precision,
    "destination_latitude" double precision,
    "destination_longitude" double precision,
    "service_type_ids" jsonb,
    "shipment_type_ids" jsonb,
    "tractor_type_ids" jsonb,
    "trailer_type_ids" jsonb,
    "commodity_ids" jsonb,
    "freight_classes" jsonb,
    "service_models" jsonb,
    "equipment_classes" jsonb,
    "min_weight" numeric(12, 2),
    "max_weight" numeric(12, 2),
    "min_distance" numeric(12, 2),
    "max_distance" numeric(12, 2),
    "min_stops" smallint,
    "max_stops" smallint,
    "days_of_week" smallint NOT NULL DEFAULT 0,
    "hazmat_only" boolean NOT NULL DEFAULT FALSE,
    "temp_control_only" boolean NOT NULL DEFAULT FALSE,
    "rating_basis" rate_agreement_basis_enum NOT NULL,
    "rate" numeric(19, 6),
    "rate_matrix_id" varchar(100),
    "formula_template_id" varchar(100),
    "percent_basis" rate_agreement_percent_basis_enum,
    "currency" varchar(3),
    "freight_class_source" rate_freight_class_source_enum NOT NULL DEFAULT 'Commodity',
    "fixed_freight_class" freight_class_enum,
    "density_scale_id" varchar(100),
    "discount_percent" numeric(9, 4),
    "absolute_min_charge" numeric(19, 4),
    "allow_deficit_rating" boolean NOT NULL DEFAULT TRUE,
    "min_charge" numeric(19, 4),
    "max_charge" numeric(19, 4),
    "min_billable_distance" numeric(12, 2),
    "rounding_mode" rate_rounding_mode_enum,
    "priority" smallint NOT NULL DEFAULT 0,
    "specificity_score" integer NOT NULL DEFAULT 0,
    "effective_from" bigint NOT NULL,
    "effective_to" bigint,
    "supersedes_rule_id" varchar(100),
    "source_import_row_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_agreement_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rules_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rules_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id") REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreement_rules_density_scale" FOREIGN KEY ("density_scale_id", "organization_id", "business_unit_id") REFERENCES "rate_density_scales"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreement_rules_formula_template" FOREIGN KEY ("formula_template_id", "organization_id", "business_unit_id") REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    -- Exactly the one pricing input the basis reads. Two would leave the engine
    -- to guess which the contract meant; none would price at nothing.
    CONSTRAINT "chk_rate_agreement_rules_matrix_basis" CHECK (("rating_basis" = 'Matrix') = ("rate_matrix_id" IS NOT NULL)),
    CONSTRAINT "chk_rate_agreement_rules_formula_basis" CHECK (("rating_basis" = 'Formula') = ("formula_template_id" IS NOT NULL)),
    CONSTRAINT "chk_rate_agreement_rules_percent_basis" CHECK ("rating_basis" <> 'Percent' OR "percent_basis" IS NOT NULL),
    CONSTRAINT "chk_rate_agreement_rules_sell_rate_basis" CHECK ("percent_basis" <> 'SellRate' OR "party_type" = 'Carrier'),
    CONSTRAINT "chk_rate_agreement_rules_fixed_class" CHECK (("freight_class_source" = 'Fixed') = ("fixed_freight_class" IS NOT NULL)),
    CONSTRAINT "chk_rate_agreement_rules_origin_radius" CHECK (("origin_scope_type" = 'Radius') = ("origin_radius_meters" IS NOT NULL)),
    CONSTRAINT "chk_rate_agreement_rules_destination_radius" CHECK (("destination_scope_type" = 'Radius') = ("destination_radius_meters" IS NOT NULL)),
    CONSTRAINT "chk_rate_agreement_rules_origin_centre" CHECK ("origin_scope_type" <> 'Radius' OR ("origin_latitude" IS NOT NULL AND "origin_longitude" IS NOT NULL)),
    CONSTRAINT "chk_rate_agreement_rules_destination_centre" CHECK ("destination_scope_type" <> 'Radius' OR ("destination_latitude" IS NOT NULL AND "destination_longitude" IS NOT NULL)),
    CONSTRAINT "chk_rate_agreement_rules_window" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from"),
    CONSTRAINT "chk_rate_agreement_rules_weight" CHECK ("min_weight" IS NULL OR "max_weight" IS NULL OR "max_weight" > "min_weight"),
    CONSTRAINT "chk_rate_agreement_rules_distance" CHECK ("min_distance" IS NULL OR "max_distance" IS NULL OR "max_distance" > "min_distance"),
    CONSTRAINT "chk_rate_agreement_rules_stops" CHECK ("min_stops" IS NULL OR "max_stops" IS NULL OR "max_stops" >= "min_stops"),
    CONSTRAINT "chk_rate_agreement_rules_charges" CHECK ("min_charge" IS NULL OR "max_charge" IS NULL OR "max_charge" >= "min_charge"),
    CONSTRAINT "chk_rate_agreement_rules_discount" CHECK ("discount_percent" IS NULL OR ("discount_percent" >= 0 AND "discount_percent" <= 100)),
    CONSTRAINT "chk_rate_agreement_rules_rate" CHECK ("rate" IS NULL OR "rate" >= 0),
    CONSTRAINT "chk_rate_agreement_rules_days" CHECK ("days_of_week" BETWEEN 0 AND 127)
);

--bun:split
-- The centres are generated rather than written so the geometry can never
-- disagree with the coordinates a user edited.
ALTER TABLE "rate_agreement_rules"
    ADD COLUMN IF NOT EXISTS "origin_center" geography(point, 4326) GENERATED ALWAYS AS (CASE WHEN "origin_latitude" IS NOT NULL AND "origin_longitude" IS NOT NULL THEN
        ST_SetSRID(ST_MakePoint("origin_longitude", "origin_latitude"), 4326)::geography
    ELSE
        NULL
    END) STORED;

--bun:split
ALTER TABLE "rate_agreement_rules"
    ADD COLUMN IF NOT EXISTS "destination_center" geography(point, 4326) GENERATED ALWAYS AS (CASE WHEN "destination_latitude" IS NOT NULL AND "destination_longitude" IS NOT NULL THEN
        ST_SetSRID(ST_MakePoint("destination_longitude", "destination_latitude"), 4326)::geography
    ELSE
        NULL
    END) STORED;

--bun:split
-- The resolution index. Leading with the party narrows to one customer or
-- carrier before the lane is even considered, which is the difference between
-- reading a handful of rows and reading every rule in the organization that
-- happens to share the lane. A shipment then produces at most a few dozen
-- candidate lane keys and the planner runs a bitmap OR of that many selective
-- probes, so the cost tracks matching rules rather than table size. The
-- included columns let the ordering and the effective window filter be answered
-- without visiting the heap.
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rules_resolve" ON "rate_agreement_rules"("organization_id", "business_unit_id", "party_type", "party_id", "lane_key", "effective_from" DESC) INCLUDE ("effective_to", "rate_agreement_id", "specificity_score", "priority")
WHERE
    "status" = 'Active';

--bun:split
-- Radius lanes cannot be reduced to a key, so they are found geospatially by a
-- second, much smaller query and unioned with the keyed results.
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rules_origin_center" ON "rate_agreement_rules" USING GIST("origin_center")
WHERE
    "origin_scope_type" = 'Radius' AND "status" = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rules_destination_center" ON "rate_agreement_rules" USING GIST("destination_center")
WHERE
    "destination_scope_type" = 'Radius' AND "status" = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rules_agreement" ON "rate_agreement_rules"("organization_id", "business_unit_id", "rate_agreement_id", "effective_from" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rules_supersedes" ON "rate_agreement_rules"("organization_id", "business_unit_id", "supersedes_rule_id")
WHERE
    "supersedes_rule_id" IS NOT NULL;

--bun:split
COMMENT ON TABLE rate_agreement_rules IS 'One priced lane. lane_key and specificity_score are derived on write so the values the database indexes and orders by cannot drift from the fields a person edited';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_agreement_rule_breaks"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_agreement_rule_id" varchar(100) NOT NULL,
    "from_weight" numeric(12, 2) NOT NULL,
    "to_weight" numeric(12, 2),
    "rate" numeric(19, 6) NOT NULL,
    "min_charge" numeric(19, 4),
    "label" varchar(50),
    "sort_order" smallint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_agreement_rule_breaks" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_rule_breaks_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rule_breaks_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rule_breaks_rule" FOREIGN KEY ("rate_agreement_rule_id", "organization_id", "business_unit_id") REFERENCES "rate_agreement_rules"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_agreement_rule_breaks_from" CHECK ("from_weight" >= 0),
    CONSTRAINT "chk_rate_agreement_rule_breaks_band" CHECK ("to_weight" IS NULL OR "to_weight" > "from_weight"),
    CONSTRAINT "chk_rate_agreement_rule_breaks_rate" CHECK ("rate" >= 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_rule_breaks_from" ON "rate_agreement_rule_breaks"("organization_id", "business_unit_id", "rate_agreement_rule_id", "from_weight");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rule_breaks_rule" ON "rate_agreement_rule_breaks"("organization_id", "business_unit_id", "rate_agreement_rule_id", "from_weight");

--bun:split
CREATE TABLE IF NOT EXISTS "rate_agreement_accessorials"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_agreement_id" varchar(100) NOT NULL,
    "accessorial_charge_id" varchar(100) NOT NULL,
    "method" accessorial_method_enum NOT NULL,
    "rate_unit" rate_unit_enum,
    "amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "waived" boolean NOT NULL DEFAULT FALSE,
    "auto_apply" boolean NOT NULL DEFAULT FALSE,
    "apply_condition" text,
    "free_units" smallint,
    "max_amount" numeric(19, 4),
    "formula_template_id" varchar(100),
    "service_type_ids" jsonb,
    "shipment_type_ids" jsonb,
    "effective_from" bigint,
    "effective_to" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_agreement_accessorials" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_accessorials_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_accessorials_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_accessorials_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_accessorials_charge" FOREIGN KEY ("accessorial_charge_id", "organization_id", "business_unit_id") REFERENCES "accessorial_charges"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreement_accessorials_formula_template" FOREIGN KEY ("formula_template_id", "organization_id", "business_unit_id") REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_agreement_accessorials_amount" CHECK ("amount" >= 0),
    CONSTRAINT "chk_rate_agreement_accessorials_waived" CHECK (NOT "waived" OR "amount" = 0),
    CONSTRAINT "chk_rate_agreement_accessorials_rate_unit" CHECK ("method" <> 'PerUnit' OR "rate_unit" IS NOT NULL),
    CONSTRAINT "chk_rate_agreement_accessorials_free_units" CHECK ("free_units" IS NULL OR "free_units" >= 0),
    CONSTRAINT "chk_rate_agreement_accessorials_condition" CHECK ("apply_condition" IS NULL OR "auto_apply"),
    CONSTRAINT "chk_rate_agreement_accessorials_window" CHECK ("effective_from" IS NULL OR "effective_to" IS NULL OR "effective_to" > "effective_from")
);

--bun:split
-- Two schedule rows for one accessorial would each try to price the same
-- charge, and which won would depend on row order.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_accessorials" ON "rate_agreement_accessorials"("organization_id", "business_unit_id", "rate_agreement_id", "accessorial_charge_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_agreement_accessorials_auto" ON "rate_agreement_accessorials"("organization_id", "business_unit_id", "rate_agreement_id")
WHERE
    "auto_apply";

--bun:split
COMMENT ON TABLE rate_agreement_accessorials IS 'A contract price for an accessorial, overriding the organization default; this is what lets the rate confirmation and the invoice agree';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_agreement_fuel_bindings"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "rate_agreement_id" varchar(100) NOT NULL,
    "fuel_surcharge_program_id" varchar(100) NOT NULL,
    "waived" boolean NOT NULL DEFAULT FALSE,
    "peg_price_override" numeric(19, 4),
    "increment_rate_override" numeric(19, 4),
    "cap_amount" numeric(19, 4),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_agreement_fuel_bindings" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_fuel_bindings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_fuel_bindings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_fuel_bindings_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_fuel_bindings_program" FOREIGN KEY ("fuel_surcharge_program_id", "organization_id", "business_unit_id") REFERENCES "fuel_surcharge_programs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_agreement_fuel_bindings_waived" CHECK (NOT "waived" OR ("peg_price_override" IS NULL AND "increment_rate_override" IS NULL AND "cap_amount" IS NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_fuel_bindings" ON "rate_agreement_fuel_bindings"("organization_id", "business_unit_id", "rate_agreement_id");

--bun:split
COMMENT ON TABLE rate_agreement_fuel_bindings IS 'Contract level fuel terms; resolution order is agreement, then customer billing profile, then organization default';

--bun:split
CREATE TABLE IF NOT EXISTS "rate_quotes"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "shipment_id" varchar(100),
    "shipment_move_id" varchar(100),
    "party_type" rate_agreement_party_type_enum NOT NULL,
    "party_id" varchar(100) NOT NULL,
    "purpose" rate_quote_purpose_enum NOT NULL DEFAULT 'Rating',
    "outcome" rate_quote_outcome_enum NOT NULL,
    "status" rate_quote_status_enum NOT NULL DEFAULT 'Applied',
    "rate_agreement_id" varchar(100),
    "rate_agreement_rule_id" varchar(100),
    "agreement_version_number" bigint,
    "formula_template_id" varchar(100),
    "specificity_score" integer NOT NULL DEFAULT 0,
    "currency" varchar(3) NOT NULL,
    "linehaul_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "fuel_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "accessorial_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "total_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "cost_amount" numeric(19, 4),
    "margin_amount" numeric(19, 4),
    "margin_percent" numeric(9, 4),
    "billing_currency" varchar(3) NOT NULL,
    "billing_amount" numeric(19, 4) NOT NULL DEFAULT 0,
    "fx_rate" numeric(24, 12),
    "exchange_rate_id" varchar(100),
    "as_of" bigint NOT NULL,
    "rated_at" bigint NOT NULL,
    "rated_by_id" varchar(100),
    "engine_version" varchar(32) NOT NULL,
    "context_hash" varchar(64) NOT NULL,
    "override_reason" text,
    "foregone_amount" numeric(19, 4),
    "trace" jsonb,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_rate_quotes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_quotes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_quotes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_quotes_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    -- Agreements are archived rather than deleted precisely because quotes
    -- point at them, so the reference is restricted rather than cascading.
    CONSTRAINT "fk_rate_quotes_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_quotes_rated_names_agreement" CHECK ("outcome" <> 'Rated' OR "rate_agreement_id" IS NOT NULL),
    CONSTRAINT "chk_rate_quotes_fallback_names_template" CHECK ("outcome" <> 'FormulaFallback' OR "formula_template_id" IS NOT NULL)
);

--bun:split
-- One quote governs a shipment at a time; the rest are the history a dispute
-- reads.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_quotes_applied" ON "rate_quotes"("organization_id", "business_unit_id", "shipment_id", "party_type")
WHERE
    "status" = 'Applied' AND "shipment_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_quotes_shipment" ON "rate_quotes"("organization_id", "business_unit_id", "shipment_id", "rated_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_quotes_party" ON "rate_quotes"("organization_id", "business_unit_id", "party_type", "party_id", "rated_at" DESC);

--bun:split
-- Answers "which rules have never won" and "what does margin look like on this
-- contract", which a blob on the shipment could not.
CREATE INDEX IF NOT EXISTS "idx_rate_quotes_rule" ON "rate_quotes"("organization_id", "business_unit_id", "rate_agreement_rule_id", "rated_at" DESC)
WHERE
    "rate_agreement_rule_id" IS NOT NULL;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_quotes_attention" ON "rate_quotes"("organization_id", "business_unit_id", "rated_at" DESC)
WHERE
    "outcome" IN ('NoRateFound', 'Error');

--bun:split
CREATE INDEX IF NOT EXISTS "idx_rate_quotes_dates_brin" ON "rate_quotes" USING BRIN(created_at, rated_at) WITH (pages_per_range = 128);

--bun:split
COMMENT ON TABLE rate_quotes IS 'What a rating decided and why. Written on every rating including fallbacks and failures, and never overwritten, because re-rating happens on every move edit, assignment and fuel price job';

--bun:split
ALTER TABLE "shipments"
    ADD COLUMN IF NOT EXISTS "rate_quote_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "rate_agreement_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "rate_agreement_rule_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "rate_override_amount" numeric(19, 4),
    ADD COLUMN IF NOT EXISTS "rate_override_reason" text,
    ADD COLUMN IF NOT EXISTS "rate_override_by_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "rate_override_at" bigint,
    ADD COLUMN IF NOT EXISTS "rate_locked" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN "shipments"."rate_override_amount" IS 'A linehaul set by hand. Re-rating preserves it and records what the contract would have charged, which is the rate leakage report';

--bun:split
COMMENT ON COLUMN "shipments"."rate_locked" IS 'Suppresses re-rating entirely, for shipments already invoiced';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_shipments_rate_agreement" ON "shipments"("organization_id", "business_unit_id", "rate_agreement_id")
WHERE
    "rate_agreement_id" IS NOT NULL;

--bun:split
-- The formula template stops being mandatory: a shipment now needs either a
-- resolved quote or a template, and only the service layer can tell which,
-- because it depends on whether an agreement matched.
ALTER TABLE "shipments"
    ALTER COLUMN "formula_template_id" DROP NOT NULL;

--bun:split
ALTER TABLE "additional_charges"
    ADD COLUMN IF NOT EXISTS "rate_agreement_accessorial_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "rate_quote_id" varchar(100);

--bun:split
-- Three engines now write system charges. Keying each to its own owner column
-- and insisting on exactly one is what makes double billing impossible rather
-- than merely unlikely.
ALTER TABLE "additional_charges"
    ADD CONSTRAINT "chk_additional_charges_single_owner" CHECK (NOT "is_system_generated" OR (("fuel_surcharge_program_id" IS NOT NULL)::int + ("detention_occurrence_id" IS NOT NULL)::int + ("rate_agreement_accessorial_id" IS NOT NULL)::int) = 1);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_additional_charges_agreement_accessorial" ON "additional_charges"("organization_id", "business_unit_id", "rate_agreement_accessorial_id")
WHERE
    "rate_agreement_accessorial_id" IS NOT NULL;

--bun:split
-- Billing control already owns rate validation enforcement and the variance
-- tolerance, so the unrated disposition belongs beside them rather than in a
-- new control table.
ALTER TABLE "billing_controls"
    ADD COLUMN IF NOT EXISTS "unrated_shipment_disposition" unrated_shipment_disposition_enum NOT NULL DEFAULT 'FallbackFormulaTemplate',
    ADD COLUMN IF NOT EXISTS "fallback_formula_template_id" varchar(100),
    ADD COLUMN IF NOT EXISTS "require_rate_override_reason" boolean NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS "enforce_margin_floor" boolean NOT NULL DEFAULT FALSE;

--bun:split
COMMENT ON COLUMN "billing_controls"."unrated_shipment_disposition" IS 'What happens when no agreement covers a lane. The default preserves the behaviour that existed before rate agreements: fall back to the formula template on the shipment';
