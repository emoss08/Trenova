CREATE TYPE "mode_profile_status_enum" AS ENUM(
    'Draft',
    'Active',
    'Archived'
);

--bun:split
CREATE TYPE "mode_profile_service_model_enum" AS ENUM(
    'Truckload',
    'Dedicated',
    'LTL',
    'Expedite',
    'FinalMile',
    'Drayage'
);

--bun:split
CREATE TYPE "mode_profile_equipment_class_enum" AS ENUM(
    'DryVan',
    'Refrigerated',
    'Flatbed',
    'Tank',
    'DryBulk',
    'Container',
    'AutoRack',
    'StraightTruck'
);

--bun:split
CREATE TYPE "mode_profile_execution_party_enum" AS ENUM(
    'CompanyAsset',
    'OwnerOperator',
    'LeasedOn',
    'PurchasedCapacity'
);

--bun:split
CREATE TYPE "mode_profile_deviation_state_enum" AS ENUM(
    'Open',
    'Acknowledged',
    'Resolved'
);

--bun:split
CREATE TABLE IF NOT EXISTS "mode_profiles"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "code" varchar(50) NOT NULL,
    "description" text,
    "status" mode_profile_status_enum NOT NULL DEFAULT 'Draft',
    "service_model" mode_profile_service_model_enum NOT NULL,
    "equipment_class" mode_profile_equipment_class_enum NOT NULL,
    "execution_party" mode_profile_execution_party_enum NOT NULL DEFAULT 'CompanyAsset',
    "capabilities" jsonb,
    "is_org_default" boolean NOT NULL DEFAULT FALSE,
    "priority" smallint NOT NULL DEFAULT 0,
    "specificity_score" integer NOT NULL DEFAULT 0,
    "customer_id" varchar(100),
    "shipment_type_ids" jsonb,
    "service_type_ids" jsonb,
    "equipment_type_ids" jsonb,
    "effective_start_date" bigint,
    "effective_end_date" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_mode_profiles" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_mode_profiles_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profiles_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profiles_customer" FOREIGN KEY ("customer_id", "organization_id", "business_unit_id") REFERENCES "customers"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
    CONSTRAINT "chk_mode_profiles_effective_window" CHECK ("effective_start_date" IS NULL OR "effective_end_date" IS NULL OR "effective_end_date" > "effective_start_date"),
    CONSTRAINT "chk_mode_profiles_org_default_scope" CHECK (NOT "is_org_default" OR ("customer_id" IS NULL AND "shipment_type_ids" IS NULL AND "service_type_ids" IS NULL AND "equipment_type_ids" IS NULL))
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_mode_profiles_code" ON "mode_profiles"("organization_id", "business_unit_id", lower("code"));

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_mode_profiles_org_default" ON "mode_profiles"("organization_id", "business_unit_id")
WHERE
    "is_org_default";

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profiles_resolution" ON "mode_profiles"("organization_id", "business_unit_id", "priority" DESC, "specificity_score" DESC, "created_at")
WHERE
    "status" = 'Active';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profiles_customer" ON "mode_profiles"("customer_id")
WHERE
    "customer_id" IS NOT NULL;

--bun:split
ALTER TABLE "mode_profiles"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("code", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("name", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("description", '')), 'B')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profiles_search" ON "mode_profiles" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE mode_profiles IS 'Capability presets composing service model, equipment class, and execution party; resolved per shipment to drive validation, field requirements, and UI disclosure';

--bun:split
CREATE TABLE IF NOT EXISTS "mode_profile_capability_rules"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "mode_profile_id" varchar(100) NOT NULL,
    "rule_key" varchar(100) NOT NULL,
    "capability" varchar(100) NOT NULL,
    "enforcement" enforcement_level_enum NOT NULL DEFAULT 'Block',
    "enabled" boolean NOT NULL DEFAULT TRUE,
    "parameters" jsonb,
    "override_reason" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_mode_profile_capability_rules" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_mode_profile_capability_rules_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_capability_rules_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_capability_rules_profile" FOREIGN KEY ("mode_profile_id", "organization_id", "business_unit_id") REFERENCES "mode_profiles"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_mode_profile_capability_rules_key" ON "mode_profile_capability_rules"("mode_profile_id", "organization_id", "business_unit_id", "rule_key");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profile_capability_rules_profile" ON "mode_profile_capability_rules"("mode_profile_id", "capability");

--bun:split
COMMENT ON TABLE mode_profile_capability_rules IS 'Per-profile rule enforcement levels; Ignore, Warn, RequireReview, and Block replace boolean feature flags so every relaxation is explicit and explainable';

--bun:split
CREATE TABLE IF NOT EXISTS "mode_profile_deviations"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "mode_profile_id" varchar(100) NOT NULL,
    "capability_rule_id" varchar(100),
    "rule_key" varchar(100) NOT NULL,
    "capability" varchar(100) NOT NULL,
    "enforcement" enforcement_level_enum NOT NULL,
    "resource_type" varchar(100) NOT NULL,
    "resource_id" varchar(100) NOT NULL,
    "field" varchar(200),
    "message" text NOT NULL,
    "state" mode_profile_deviation_state_enum NOT NULL DEFAULT 'Open',
    "occurred_at" bigint NOT NULL,
    "acknowledged_by_id" varchar(100),
    "acknowledged_at" bigint,
    "acknowledgement_reason" text,
    "provenance" jsonb,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_mode_profile_deviations" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_mode_profile_deviations_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_deviations_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_deviations_profile" FOREIGN KEY ("mode_profile_id", "organization_id", "business_unit_id") REFERENCES "mode_profiles"("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_mode_profile_deviations_acknowledged_by" FOREIGN KEY ("acknowledged_by_id") REFERENCES "users"("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "chk_mode_profile_deviations_recordable" CHECK ("enforcement" IN ('Warn', 'RequireReview')),
    CONSTRAINT "chk_mode_profile_deviations_acknowledgement" CHECK ("state" = 'Open' OR ("acknowledged_by_id" IS NOT NULL AND "acknowledgement_reason" IS NOT NULL AND char_length("acknowledgement_reason") >= 10))
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profile_deviations_resource" ON "mode_profile_deviations"("organization_id", "business_unit_id", "resource_type", "resource_id", "occurred_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profile_deviations_open" ON "mode_profile_deviations"("organization_id", "business_unit_id", "rule_key", "occurred_at" DESC)
WHERE
    "state" = 'Open';

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profile_deviations_ledger" ON "mode_profile_deviations"("organization_id", "business_unit_id", "mode_profile_id", "rule_key", "occurred_at" DESC);

--bun:split
ALTER TABLE "mode_profile_deviations"
    ADD COLUMN IF NOT EXISTS search_vector tsvector GENERATED ALWAYS AS (setweight(immutable_to_tsvector('simple', COALESCE("rule_key", '')), 'A') || setweight(immutable_to_tsvector('simple', COALESCE("message", '')), 'B') || setweight(immutable_to_tsvector('simple', COALESCE("acknowledgement_reason", '')), 'C')) STORED;

--bun:split
CREATE INDEX IF NOT EXISTS "idx_mode_profile_deviations_search" ON "mode_profile_deviations" USING GIN(search_vector);

--bun:split
COMMENT ON TABLE mode_profile_deviations IS 'Deviation ledger: every warn or review level rule result, the shipment it fired on, and the reason a user accepted it';

--bun:split
INSERT INTO "mode_profiles"("id", "business_unit_id", "organization_id", "name", "code", "description", "status", "service_model", "equipment_class", "execution_party", "capabilities", "is_org_default", "priority", "specificity_score")
SELECT
    'mpf_' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 26)),
    sc."business_unit_id",
    sc."organization_id",
    'OTR Truckload',
    'OTR',
    'Default capability profile derived from this organization''s existing shipment control settings.',
    'Active',
    'Truckload',
    'DryVan',
    'CompanyAsset',
    '["Core", "Hazmat", "TemperatureControl"]'::jsonb,
    TRUE,
    0,
    0
FROM
    "shipment_controls" sc
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "mode_profile_capability_rules"("id", "business_unit_id", "organization_id", "mode_profile_id", "rule_key", "capability", "enforcement", "enabled", "parameters", "override_reason")
SELECT
    'mpr_' || upper(substr(replace(gen_random_uuid()::text, '-', ''), 1, 26)),
    mp."business_unit_id",
    mp."organization_id",
    mp."id",
    r."rule_key",
    r."capability",
    r."enforcement",
    TRUE,
    r."parameters",
    r."override_reason"
FROM
    "mode_profiles" mp
    JOIN "shipment_controls" sc ON sc."organization_id" = mp."organization_id"
        AND sc."business_unit_id" = mp."business_unit_id"
    CROSS JOIN LATERAL (
        VALUES ('cargo.maxShipmentWeight',
            'Core',
            'Block'::enforcement_level_enum,
            jsonb_build_object('maxWeight', sc."max_shipment_weight_limit"),
            NULL),
        ('dispatch.moveRemoval',
            'Core',
            (CASE WHEN sc."allow_move_removals" THEN
                'Ignore'
            ELSE
                'Block'
            END)::enforcement_level_enum,
            NULL,
            (CASE WHEN sc."allow_move_removals" THEN
                'Move removals were permitted by this organization''s shipment control settings before capability profiles were introduced.'
            ELSE
                NULL
            END)),
        ('documentation.duplicateBol',
            'Core',
            (CASE WHEN sc."check_for_duplicate_bols" THEN
                'Block'
            ELSE
                'Ignore'
            END)::enforcement_level_enum,
            NULL,
            (CASE WHEN sc."check_for_duplicate_bols" THEN
                NULL
            ELSE
                'Duplicate bill of lading checking was disabled in this organization''s shipment control settings before capability profiles were introduced.'
            END)),
        ('hazmat.segregation',
            'Hazmat',
            (CASE WHEN sc."check_hazmat_segregation" THEN
                'Block'
            ELSE
                'Ignore'
            END)::enforcement_level_enum,
            NULL,
            (CASE WHEN sc."check_hazmat_segregation" THEN
                NULL
            ELSE
                'Hazmat segregation checking was disabled in this organization''s shipment control settings before capability profiles were introduced.'
            END)),
        ('cargo.temperatureRange',
            'TemperatureControl',
            'Ignore'::enforcement_level_enum,
            NULL,
            'Temperature range enforcement is new in capability profiles and starts disabled so existing shipments are unaffected. Raise it to Warn or Block once reefer lanes are reviewed.')) AS r("rule_key",
            "capability",
            "enforcement",
            "parameters",
            "override_reason")
WHERE
    mp."is_org_default"
ON CONFLICT
    DO NOTHING;
