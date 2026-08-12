-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20260904000000_mode_profiles.tx.up.sql

CREATE TABLE IF NOT EXISTS "mode_profiles"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "code" TEXT NOT NULL,
    "description" TEXT,
    "status" TEXT NOT NULL DEFAULT 'Draft',
    "service_model" TEXT NOT NULL,
    "equipment_class" TEXT NOT NULL,
    "execution_party" TEXT NOT NULL DEFAULT 'CompanyAsset',
    "capabilities" TEXT,
    "is_org_default" INTEGER NOT NULL DEFAULT 0,
    "priority" INTEGER NOT NULL DEFAULT 0,
    "specificity_score" INTEGER NOT NULL DEFAULT 0,
    "customer_id" TEXT,
    "shipment_type_ids" TEXT,
    "service_type_ids" TEXT,
    "equipment_type_ids" TEXT,
    "effective_start_date" INTEGER,
    "effective_end_date" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_mode_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_mode_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_mode_profiles_effective_window" CHECK ("effective_start_date" IS NULL OR "effective_end_date" IS NULL OR "effective_end_date" > "effective_start_date"),
    CONSTRAINT "chk_mode_profiles_org_default_scope" CHECK (NOT "is_org_default" OR ("customer_id" IS NULL AND "shipment_type_ids" IS NULL AND "service_type_ids" IS NULL AND "equipment_type_ids" IS NULL))
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_mode_profiles_code" ON "mode_profiles" ("organization_id", "business_unit_id", lower("code"));

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_mode_profiles_org_default" ON "mode_profiles" ("organization_id", "business_unit_id")WHERE
    "is_org_default";

--bun:split

CREATE INDEX IF NOT EXISTS "idx_mode_profiles_resolution" ON "mode_profiles" ("organization_id", "business_unit_id", "priority" DESC, "specificity_score" DESC, "created_at")WHERE
    "status" = 'Active';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_mode_profiles_customer" ON "mode_profiles" ("customer_id")WHERE
    "customer_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "mode_profile_capability_rules"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "mode_profile_id" TEXT NOT NULL,
    "rule_key" TEXT NOT NULL,
    "capability" TEXT NOT NULL,
    "enforcement" TEXT NOT NULL DEFAULT 'Block',
    "enabled" INTEGER NOT NULL DEFAULT 1,
    "parameters" TEXT,
    "override_reason" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_mode_profile_capability_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_mode_profile_capability_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_capability_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_capability_rules_profile" FOREIGN KEY ("mode_profile_id", "organization_id", "business_unit_id") REFERENCES "mode_profiles"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_mode_profile_capability_rules_key" ON "mode_profile_capability_rules" ("mode_profile_id", "organization_id", "business_unit_id", "rule_key");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_mode_profile_capability_rules_profile" ON "mode_profile_capability_rules" ("mode_profile_id", "capability");

--bun:split

CREATE TABLE IF NOT EXISTS "mode_profile_deviations"(
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "mode_profile_id" TEXT NOT NULL,
    "capability_rule_id" TEXT,
    "rule_key" TEXT NOT NULL,
    "capability" TEXT NOT NULL,
    "enforcement" TEXT NOT NULL,
    "resource_type" TEXT NOT NULL,
    "resource_id" TEXT NOT NULL,
    "field" TEXT,
    "message" TEXT NOT NULL,
    "state" TEXT NOT NULL DEFAULT 'Open',
    "occurred_at" INTEGER NOT NULL,
    "acknowledged_by_id" TEXT,
    "acknowledged_at" INTEGER,
    "acknowledgement_reason" TEXT,
    "provenance" TEXT,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_mode_profile_deviations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_mode_profile_deviations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_deviations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_deviations_profile" FOREIGN KEY ("mode_profile_id", "organization_id", "business_unit_id") REFERENCES "mode_profiles"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_deviations_acknowledged_by" FOREIGN KEY ("acknowledged_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_mode_profile_deviations_recordable" CHECK ("enforcement" IN ('Warn', 'RequireReview')),
    CONSTRAINT "chk_mode_profile_deviations_acknowledgement" CHECK ("state" = 'Open' OR ("acknowledged_by_id" IS NOT NULL AND "acknowledgement_reason" IS NOT NULL AND length("acknowledgement_reason") >= 10))
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_mode_profile_deviations_resource" ON "mode_profile_deviations" ("organization_id", "business_unit_id", "resource_type", "resource_id", "occurred_at" DESC);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_mode_profile_deviations_open" ON "mode_profile_deviations" ("organization_id", "business_unit_id", "rule_key", "occurred_at" DESC)WHERE
    "state" = 'Open';

--bun:split

CREATE INDEX IF NOT EXISTS "idx_mode_profile_deviations_ledger" ON "mode_profile_deviations" ("organization_id", "business_unit_id", "mode_profile_id", "rule_key", "occurred_at" DESC);
