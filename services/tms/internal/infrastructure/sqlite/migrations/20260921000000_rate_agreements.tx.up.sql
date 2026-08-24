-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260921000000_rate_agreements.tx.up.sql

CREATE TABLE IF NOT EXISTS "rate_agreements"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "party_type" TEXT NOT NULL,
    "customer_id" TEXT,
    "carrier_id" TEXT,
    "code" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "agreement_type" TEXT NOT NULL DEFAULT 'Contract',
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "contract_ref" TEXT,
    "document_id" TEXT,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "effective_from" INTEGER NOT NULL,
    "effective_to" INTEGER,
    "auto_renew" INTEGER NOT NULL DEFAULT 0,
    "renewal_notice_days" INTEGER NOT NULL DEFAULT 30,
    "currency" TEXT NOT NULL DEFAULT 'USD',
    "default_min_charge" REAL,
    "default_max_charge" REAL,
    "rounding_mode" TEXT NOT NULL DEFAULT 'HalfUp',
    "rounding_precision" INTEGER NOT NULL DEFAULT 2,
    "bill_to_customer_id" TEXT,
    "margin_floor_percent" REAL,
    "max_pay_percent_of_sell" REAL,
    "submitted_by_id" TEXT,
    "submitted_at" INTEGER,
    "approved_by_id" TEXT,
    "approved_at" INTEGER,
    "review_comment" TEXT,
    "current_version_number" INTEGER NOT NULL DEFAULT 1,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_agreements" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreements_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreements_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreements_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreements_bill_to" FOREIGN KEY ("bill_to_customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreements_carrier" FOREIGN KEY ("carrier_id", "organization_id", "business_unit_id") REFERENCES "carriers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_agreements_party" CHECK (("party_type" = 'Customer' AND "customer_id" IS NOT NULL AND "carrier_id" IS NULL) OR ("party_type" = 'Carrier' AND "carrier_id" IS NOT NULL AND "customer_id" IS NULL)),
    CONSTRAINT "chk_rate_agreements_sell_side_fields" CHECK ("party_type" = 'Customer' OR "bill_to_customer_id" IS NULL),
    CONSTRAINT "chk_rate_agreements_buy_side_fields" CHECK ("party_type" = 'Carrier' OR ("margin_floor_percent" IS NULL AND "max_pay_percent_of_sell" IS NULL)),
    CONSTRAINT "chk_rate_agreements_window" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from"),
    CONSTRAINT "chk_rate_agreements_charges" CHECK ("default_min_charge" IS NULL OR "default_max_charge" IS NULL OR "default_max_charge" >= "default_min_charge"),
    CONSTRAINT "chk_rate_agreements_precision" CHECK ("rounding_precision" BETWEEN 0 AND 6)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreements_code" ON "rate_agreements" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_agreements_customer" ON "rate_agreements" ("organization_id", "business_unit_id", "customer_id", "effective_from" DESC)WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_agreements_carrier" ON "rate_agreements" ("organization_id", "business_unit_id", "carrier_id", "effective_from" DESC)WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_agreements_expiring" ON "rate_agreements" ("organization_id", "business_unit_id", "effective_to")WHERE
    "status" = 'Active' AND "effective_to" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "rate_agreement_versions"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_agreement_id" TEXT NOT NULL,
    "version_number" INTEGER NOT NULL,
    "effective_from" INTEGER NOT NULL,
    "effective_to" INTEGER,
    "name" TEXT NOT NULL,
    "description" TEXT,
    "agreement_type" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "contract_ref" TEXT,
    "currency" TEXT NOT NULL,
    "default_min_charge" REAL,
    "default_max_charge" REAL,
    "rounding_mode" TEXT NOT NULL,
    "rounding_precision" INTEGER NOT NULL,
    "margin_floor_percent" REAL,
    "max_pay_percent_of_sell" REAL,
    "change_message" TEXT,
    "change_summary" TEXT,
    "created_by_id" TEXT NOT NULL,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_agreement_versions" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_versions_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_versions_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_versions_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_versions_created_by" FOREIGN KEY ("created_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_agreement_versions_window" CHECK ("effective_to" IS NULL OR "effective_to" > "effective_from")
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_versions_number" ON "rate_agreement_versions" ("organization_id", "business_unit_id", "rate_agreement_id", "version_number");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_agreement_versions_effective" ON "rate_agreement_versions" ("organization_id", "business_unit_id", "rate_agreement_id", "effective_from" DESC);

--bun:split

CREATE TABLE IF NOT EXISTS "rate_agreement_rules"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_agreement_id" TEXT NOT NULL,
    "party_type" TEXT NOT NULL,
    "party_id" TEXT NOT NULL,
    "label" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Active',
    "origin_scope_type" TEXT NOT NULL,
    "origin_scope_value" TEXT,
    "origin_city" TEXT,
    "destination_scope_type" TEXT NOT NULL,
    "destination_scope_value" TEXT,
    "destination_city" TEXT,
    "lane_key" TEXT NOT NULL,
    "direction" TEXT NOT NULL DEFAULT 'Directional',
    "origin_radius_meters" REAL,
    "destination_radius_meters" REAL,
    "origin_latitude" REAL,
    "origin_longitude" REAL,
    "destination_latitude" REAL,
    "destination_longitude" REAL,
    "service_type_ids" TEXT,
    "shipment_type_ids" TEXT,
    "tractor_type_ids" TEXT,
    "trailer_type_ids" TEXT,
    "commodity_ids" TEXT,
    "freight_classes" TEXT,
    "service_models" TEXT,
    "equipment_classes" TEXT,
    "min_weight" REAL,
    "max_weight" REAL,
    "min_distance" REAL,
    "max_distance" REAL,
    "min_stops" INTEGER,
    "max_stops" INTEGER,
    "days_of_week" INTEGER NOT NULL DEFAULT 0,
    "hazmat_only" INTEGER NOT NULL DEFAULT 0,
    "temp_control_only" INTEGER NOT NULL DEFAULT 0,
    "rating_basis" TEXT NOT NULL,
    "rate" REAL,
    "rate_matrix_id" TEXT,
    "formula_template_id" TEXT,
    "percent_basis" TEXT,
    "currency" TEXT,
    "freight_class_source" TEXT NOT NULL DEFAULT 'Commodity',
    "fixed_freight_class" TEXT,
    "density_scale_id" TEXT,
    "discount_percent" REAL,
    "absolute_min_charge" REAL,
    "allow_deficit_rating" INTEGER NOT NULL DEFAULT 1,
    "min_charge" REAL,
    "max_charge" REAL,
    "min_billable_distance" REAL,
    "rounding_mode" TEXT,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "specificity_score" INTEGER NOT NULL DEFAULT 0,
    "effective_from" INTEGER NOT NULL,
    "effective_to" INTEGER,
    "supersedes_rule_id" TEXT,
    "source_import_row_id" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_agreement_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rules_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rules_matrix" FOREIGN KEY ("rate_matrix_id", "organization_id", "business_unit_id") REFERENCES "rate_matrices"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreement_rules_density_scale" FOREIGN KEY ("density_scale_id", "organization_id", "business_unit_id") REFERENCES "rate_density_scales"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "fk_rate_agreement_rules_formula_template" FOREIGN KEY ("formula_template_id", "organization_id", "business_unit_id") REFERENCES "formula_templates"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
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

CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rules_agreement" ON "rate_agreement_rules" ("organization_id", "business_unit_id", "rate_agreement_id", "effective_from" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rules_supersedes" ON "rate_agreement_rules" ("organization_id", "business_unit_id", "supersedes_rule_id")WHERE
    "supersedes_rule_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "rate_agreement_rule_breaks"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_agreement_rule_id" TEXT NOT NULL,
    "from_weight" REAL NOT NULL,
    "to_weight" REAL,
    "rate" REAL NOT NULL,
    "min_charge" REAL,
    "label" TEXT,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_agreement_rule_breaks" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_rule_breaks_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rule_breaks_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_rule_breaks_rule" FOREIGN KEY ("rate_agreement_rule_id", "organization_id", "business_unit_id") REFERENCES "rate_agreement_rules"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_rate_agreement_rule_breaks_from" CHECK ("from_weight" >= 0),
    CONSTRAINT "chk_rate_agreement_rule_breaks_band" CHECK ("to_weight" IS NULL OR "to_weight" > "from_weight"),
    CONSTRAINT "chk_rate_agreement_rule_breaks_rate" CHECK ("rate" >= 0)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_rule_breaks_from" ON "rate_agreement_rule_breaks" ("organization_id", "business_unit_id", "rate_agreement_rule_id", "from_weight");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_agreement_rule_breaks_rule" ON "rate_agreement_rule_breaks" ("organization_id", "business_unit_id", "rate_agreement_rule_id", "from_weight");

--bun:split

CREATE TABLE IF NOT EXISTS "rate_agreement_accessorials"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_agreement_id" TEXT NOT NULL,
    "accessorial_charge_id" TEXT NOT NULL,
    "method" TEXT NOT NULL,
    "rate_unit" TEXT,
    "amount" REAL NOT NULL DEFAULT 0,
    "waived" INTEGER NOT NULL DEFAULT 0,
    "auto_apply" INTEGER NOT NULL DEFAULT 0,
    "apply_condition" TEXT,
    "free_units" INTEGER,
    "max_amount" REAL,
    "formula_template_id" TEXT,
    "service_type_ids" TEXT,
    "shipment_type_ids" TEXT,
    "effective_from" INTEGER,
    "effective_to" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_accessorials" ON "rate_agreement_accessorials" ("organization_id", "business_unit_id", "rate_agreement_id", "accessorial_charge_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_agreement_accessorials_auto" ON "rate_agreement_accessorials" ("organization_id", "business_unit_id", "rate_agreement_id")WHERE
    "auto_apply";

--bun:split

CREATE TABLE IF NOT EXISTS "rate_agreement_fuel_bindings"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "rate_agreement_id" TEXT NOT NULL,
    "fuel_surcharge_program_id" TEXT NOT NULL,
    "waived" INTEGER NOT NULL DEFAULT 0,
    "peg_price_override" REAL,
    "increment_rate_override" REAL,
    "cap_amount" REAL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_agreement_fuel_bindings" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_agreement_fuel_bindings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_fuel_bindings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_fuel_bindings_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_agreement_fuel_bindings_program" FOREIGN KEY ("fuel_surcharge_program_id", "organization_id", "business_unit_id") REFERENCES "fuel_surcharge_programs"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_agreement_fuel_bindings_waived" CHECK (NOT "waived" OR ("peg_price_override" IS NULL AND "increment_rate_override" IS NULL AND "cap_amount" IS NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_agreement_fuel_bindings" ON "rate_agreement_fuel_bindings" ("organization_id", "business_unit_id", "rate_agreement_id");

--bun:split

CREATE TABLE IF NOT EXISTS "rate_quotes"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "shipment_id" TEXT,
    "shipment_move_id" TEXT,
    "party_type" TEXT NOT NULL,
    "party_id" TEXT NOT NULL,
    "purpose" TEXT NOT NULL DEFAULT 'Rating',
    "outcome" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Applied',
    "rate_agreement_id" TEXT,
    "rate_agreement_rule_id" TEXT,
    "agreement_version_number" INTEGER,
    "formula_template_id" TEXT,
    "specificity_score" INTEGER NOT NULL DEFAULT 0,
    "currency" TEXT NOT NULL,
    "linehaul_amount" REAL NOT NULL DEFAULT 0,
    "fuel_amount" REAL NOT NULL DEFAULT 0,
    "accessorial_amount" REAL NOT NULL DEFAULT 0,
    "total_amount" REAL NOT NULL DEFAULT 0,
    "cost_amount" REAL,
    "margin_amount" REAL,
    "margin_percent" REAL,
    "billing_currency" TEXT NOT NULL,
    "billing_amount" REAL NOT NULL DEFAULT 0,
    "fx_rate" REAL,
    "exchange_rate_id" TEXT,
    "as_of" INTEGER NOT NULL,
    "rated_at" INTEGER NOT NULL,
    "rated_by_id" TEXT,
    "engine_version" TEXT NOT NULL,
    "context_hash" TEXT NOT NULL,
    "override_reason" TEXT,
    "foregone_amount" REAL,
    "trace" TEXT,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_rate_quotes" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_rate_quotes_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_quotes_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_quotes_shipment" FOREIGN KEY ("shipment_id", "organization_id", "business_unit_id") REFERENCES "shipments"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_rate_quotes_agreement" FOREIGN KEY ("rate_agreement_id", "organization_id", "business_unit_id") REFERENCES "rate_agreements"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_rate_quotes_rated_names_agreement" CHECK ("outcome" <> 'Rated' OR "rate_agreement_id" IS NOT NULL),
    CONSTRAINT "chk_rate_quotes_fallback_names_template" CHECK ("outcome" <> 'FormulaFallback' OR "formula_template_id" IS NOT NULL)
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_rate_quotes_applied" ON "rate_quotes" ("organization_id", "business_unit_id", "shipment_id", "party_type")WHERE
    "status" = 'Applied' AND "shipment_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_quotes_shipment" ON "rate_quotes" ("organization_id", "business_unit_id", "shipment_id", "rated_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_quotes_party" ON "rate_quotes" ("organization_id", "business_unit_id", "party_type", "party_id", "rated_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_quotes_rule" ON "rate_quotes" ("organization_id", "business_unit_id", "rate_agreement_rule_id", "rated_at" DESC)WHERE
    "rate_agreement_rule_id" IS NOT NULL;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_rate_quotes_attention" ON "rate_quotes" ("organization_id", "business_unit_id", "rated_at" DESC)WHERE
    "outcome" IN ('NoRateFound', 'Error');

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_quote_id" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_agreement_id" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_agreement_rule_id" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_override_amount" REAL;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_override_reason" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_override_by_id" TEXT;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_override_at" INTEGER;

--bun:split

ALTER TABLE "shipments" ADD COLUMN "rate_locked" INTEGER NOT NULL DEFAULT 0;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_shipments_rate_agreement" ON "shipments" ("organization_id", "business_unit_id", "rate_agreement_id")WHERE
    "rate_agreement_id" IS NOT NULL;

--bun:split

ALTER TABLE "additional_charges" ADD COLUMN "rate_agreement_accessorial_id" TEXT;

--bun:split

ALTER TABLE "additional_charges" ADD COLUMN "rate_quote_id" TEXT;

--bun:split

CREATE INDEX IF NOT EXISTS "idx_additional_charges_agreement_accessorial" ON "additional_charges" ("organization_id", "business_unit_id", "rate_agreement_accessorial_id")WHERE
    "rate_agreement_accessorial_id" IS NOT NULL;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "unrated_shipment_disposition" TEXT NOT NULL DEFAULT 'FallbackFormulaTemplate';

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "fallback_formula_template_id" TEXT;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "require_rate_override_reason" INTEGER NOT NULL DEFAULT 1;

--bun:split

ALTER TABLE "billing_controls" ADD COLUMN "enforce_margin_floor" INTEGER NOT NULL DEFAULT 0;
