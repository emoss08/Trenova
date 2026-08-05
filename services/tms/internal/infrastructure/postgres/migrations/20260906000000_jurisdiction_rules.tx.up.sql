CREATE TYPE "jurisdiction_rule_status_enum" AS ENUM('Active', 'Inactive', 'Draft');

--bun:split
CREATE TYPE "jurisdiction_verification_state_enum" AS ENUM('Unverified', 'Verified', 'Disputed');

--bun:split
CREATE TYPE "jurisdiction_escort_role_enum" AS ENUM('Front', 'Rear', 'Police', 'RouteSurvey');

--bun:split
CREATE TYPE "jurisdiction_trigger_kind_enum" AS ENUM('Width', 'Height', 'Length', 'Weight', 'Superload');

--bun:split
CREATE TABLE IF NOT EXISTS "jurisdiction_rules"(
    "id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    "status" jurisdiction_rule_status_enum NOT NULL DEFAULT 'Active',
    "max_width_feet" numeric(10, 2) NOT NULL,
    "max_height_feet" numeric(10, 2) NOT NULL,
    "max_length_feet" numeric(10, 2) NOT NULL,
    "max_weight_pounds" bigint NOT NULL,
    "superload_width_feet" numeric(10, 2),
    "superload_weight_pounds" bigint,
    "daylight_only" boolean NOT NULL DEFAULT FALSE,
    "rush_hour_restricted" boolean NOT NULL DEFAULT FALSE,
    "weekend_restricted" boolean NOT NULL DEFAULT FALSE,
    "holiday_restricted" boolean NOT NULL DEFAULT TRUE,
    "permit_lead_time_days" smallint NOT NULL DEFAULT 1,
    "permit_validity_days" smallint NOT NULL DEFAULT 5,
    "permit_base_fee" numeric(19, 4),
    "permit_per_mile_fee" numeric(19, 4),
    "source_note" text,
    "source_url" text,
    "verification_state" jurisdiction_verification_state_enum NOT NULL DEFAULT 'Unverified',
    "verified_at" bigint,
    "effective_start_date" bigint,
    "effective_end_date" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_jurisdiction_rules" PRIMARY KEY ("id"),
    CONSTRAINT "fk_jurisdiction_rules_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_jurisdiction_rules_limits" CHECK ("max_width_feet" > 0 AND "max_height_feet" > 0 AND "max_length_feet" > 0 AND "max_weight_pounds" > 0),
    CONSTRAINT "chk_jurisdiction_rules_lead_time" CHECK ("permit_lead_time_days" >= 0 AND "permit_lead_time_days" <= 60),
    CONSTRAINT "chk_jurisdiction_rules_validity" CHECK ("permit_validity_days" > 0 AND "permit_validity_days" <= 365),
    CONSTRAINT "chk_jurisdiction_rules_effective_window" CHECK ("effective_start_date" IS NULL OR "effective_end_date" IS NULL OR "effective_end_date" > "effective_start_date")
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_jurisdiction_rules_state" ON "jurisdiction_rules"("state_id", "status");

--bun:split
COMMENT ON TABLE jurisdiction_rules IS 'Global oversize and overweight reference data per jurisdiction. Seeded rows are an unverified research baseline, not legal authority; carriers confirm each row and express their own posture through jurisdiction_rule_overrides';

--bun:split
CREATE TABLE IF NOT EXISTS "jurisdiction_escort_thresholds"(
    "id" varchar(100) NOT NULL,
    "jurisdiction_rule_id" varchar(100) NOT NULL,
    "role" jurisdiction_escort_role_enum NOT NULL,
    "trigger" jurisdiction_trigger_kind_enum NOT NULL,
    "threshold_value" numeric(12, 2) NOT NULL,
    "note" text,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_jurisdiction_escort_thresholds" PRIMARY KEY ("id"),
    CONSTRAINT "fk_jurisdiction_escort_thresholds_rule" FOREIGN KEY ("jurisdiction_rule_id") REFERENCES "jurisdiction_rules"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_jurisdiction_escort_thresholds_positive" CHECK ("threshold_value" > 0)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_jurisdiction_escort_thresholds" ON "jurisdiction_escort_thresholds"("jurisdiction_rule_id", "role", "trigger");

--bun:split
CREATE TABLE IF NOT EXISTS "jurisdiction_rule_overrides"(
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "state_id" varchar(100) NOT NULL,
    "max_width_feet" numeric(10, 2),
    "max_height_feet" numeric(10, 2),
    "max_length_feet" numeric(10, 2),
    "max_weight_pounds" bigint,
    "permit_lead_time_days" smallint,
    "daylight_only" boolean,
    "holiday_restricted" boolean,
    "reason" text NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    CONSTRAINT "pk_jurisdiction_rule_overrides" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_jurisdiction_rule_overrides_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_jurisdiction_rule_overrides_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_jurisdiction_rule_overrides_state" FOREIGN KEY ("state_id") REFERENCES "us_states"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "chk_jurisdiction_rule_overrides_reason" CHECK (char_length("reason") >= 10)
);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_jurisdiction_rule_overrides_state" ON "jurisdiction_rule_overrides"("organization_id", "business_unit_id", "state_id");

--bun:split
COMMENT ON TABLE jurisdiction_rule_overrides IS 'Per-carrier tightening of the shared jurisdiction baseline; an unset column defers to jurisdiction_rules';


--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_AK',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'AK'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_AK_' || v."role",
    'jr_seed_AK',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_AK')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_AL',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'AL'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_AL_' || v."role",
    'jr_seed_AL',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_AL')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_AR',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'AR'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_AR_' || v."role",
    'jr_seed_AR',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_AR')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_AZ',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    1,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'AZ'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_AZ_' || v."role",
    'jr_seed_AZ',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_AZ')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_CA',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    3,
    5,
    90.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'CA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_CA_' || v."role",
    'jr_seed_CA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_CA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_CO',
    s."id",
    'Active',
    8.50,
    14.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'CO'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_CO_' || v."role",
    'jr_seed_CO',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_CO')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_CT',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    4,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'CT'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_CT_' || v."role",
    'jr_seed_CT',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_CT')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_DC',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    5,
    5,
    50.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'DC'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_DC_' || v."role",
    'jr_seed_DC',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_DC')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_DE',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    20.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'DE'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_DE_' || v."role",
    'jr_seed_DE',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_DE')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_FL',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    1,
    5,
    30.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'FL'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_FL_' || v."role",
    'jr_seed_FL',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.08), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_FL')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_GA',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    2,
    5,
    30.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'GA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_GA_' || v."role",
    'jr_seed_GA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_GA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_HI',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    4,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'HI'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_HI_' || v."role",
    'jr_seed_HI',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_HI')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_IA',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'IA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_IA_' || v."role",
    'jr_seed_IA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_IA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_ID',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'ID'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_ID_' || v."role",
    'jr_seed_ID',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_ID')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_IL',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    2,
    5,
    25.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'IL'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_IL_' || v."role",
    'jr_seed_IL',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_IL')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_IN',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    1,
    5,
    20.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'IN'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_IN_' || v."role",
    'jr_seed_IN',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.50), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_IN')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_KS',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'KS'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_KS_' || v."role",
    'jr_seed_KS',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_KS')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_KY',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'KY'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_KY_' || v."role",
    'jr_seed_KY',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_KY')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_LA',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'LA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_LA_' || v."role",
    'jr_seed_LA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_LA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_MA',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    5,
    5,
    35.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'MA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_MA_' || v."role",
    'jr_seed_MA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_MA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_MD',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    4,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'MD'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_MD_' || v."role",
    'jr_seed_MD',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_MD')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_ME',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'ME'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_ME_' || v."role",
    'jr_seed_ME',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_ME')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_MI',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'MI'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_MI_' || v."role",
    'jr_seed_MI',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_MI')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_MN',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'MN'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_MN_' || v."role",
    'jr_seed_MN',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_MN')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_MO',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    20.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'MO'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_MO_' || v."role",
    'jr_seed_MO',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_MO')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_MS',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'MS'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_MS_' || v."role",
    'jr_seed_MS',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_MS')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_MT',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'MT'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_MT_' || v."role",
    'jr_seed_MT',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 16.58), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_MT')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_NC',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    30.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'NC'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_NC_' || v."role",
    'jr_seed_NC',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_NC')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_ND',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    20.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'ND'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_ND_' || v."role",
    'jr_seed_ND',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_ND')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_NE',
    s."id",
    'Active',
    8.50,
    14.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'NE'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_NE_' || v."role",
    'jr_seed_NE',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_NE')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_NH',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'NH'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_NH_' || v."role",
    'jr_seed_NH',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_NH')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_NJ',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    4,
    5,
    35.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'NJ'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_NJ_' || v."role",
    'jr_seed_NJ',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_NJ')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_NM',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'NM'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_NM_' || v."role",
    'jr_seed_NM',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_NM')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_NV',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'NV'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_NV_' || v."role",
    'jr_seed_NV',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_NV')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_NY',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    4,
    5,
    40.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'NY'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_NY_' || v."role",
    'jr_seed_NY',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_NY')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_OH',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    2,
    5,
    25.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'OH'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_OH_' || v."role",
    'jr_seed_OH',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_OH')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_OK',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'OK'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_OK_' || v."role",
    'jr_seed_OK',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_OK')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_OR',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'OR'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_OR_' || v."role",
    'jr_seed_OR',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_OR')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_PA',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    4,
    5,
    35.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'PA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_PA_' || v."role",
    'jr_seed_PA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_PA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_RI',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    4,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'RI'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_RI_' || v."role",
    'jr_seed_RI',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_RI')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_SC',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'SC'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_SC_' || v."role",
    'jr_seed_SC',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_SC')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_SD',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    20.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'SD'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_SD_' || v."role",
    'jr_seed_SD',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_SD')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_TN',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'TN'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_TN_' || v."role",
    'jr_seed_TN',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_TN')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_TX',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    1,
    5,
    60.0000,
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'TX'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_TX_' || v."role",
    'jr_seed_TX',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Researched from published state oversize permit guidance; treat as a starting baseline and confirm with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 14.08), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_TX')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_UT',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    4,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'UT'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_UT_' || v."role",
    'jr_seed_UT',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_UT')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_VA',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    30.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'VA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_VA_' || v."role",
    'jr_seed_VA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_VA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_VT',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'VT'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_VT_' || v."role",
    'jr_seed_VT',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_VT')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_WA',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    TRUE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'WA'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_WA_' || v."role",
    'jr_seed_WA',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_WA')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_WI',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'WI'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_WI_' || v."role",
    'jr_seed_WI',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_WI')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_WV',
    s."id",
    'Active',
    8.50,
    13.50,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    3,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'WV'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_WV_' || v."role",
    'jr_seed_WV',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_WV')
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_rules"("id", "state_id", "status", "max_width_feet", "max_height_feet", "max_length_feet", "max_weight_pounds", "superload_width_feet", "superload_weight_pounds", "daylight_only", "rush_hour_restricted", "holiday_restricted", "permit_lead_time_days", "permit_validity_days", "permit_base_fee", "source_note", "verification_state")
SELECT
    'jr_seed_WY',
    s."id",
    'Active',
    8.50,
    14.00,
    53.00,
    80000,
    16.00,
    200000,
    TRUE,
    FALSE,
    TRUE,
    2,
    5,
    25.0000,
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.',
    'Unverified'
FROM
    "us_states" s
WHERE
    s."abbreviation" = 'WY'
ON CONFLICT
    DO NOTHING;

--bun:split
INSERT INTO "jurisdiction_escort_thresholds"("id", "jurisdiction_rule_id", "role", "trigger", "threshold_value", "note")
SELECT
    'jet_seed_WY_' || v."role",
    'jr_seed_WY',
    v."role"::jurisdiction_escort_role_enum,
    'Width'::jurisdiction_trigger_kind_enum,
    v."threshold",
    'Regional default applied during initial seeding, NOT researched for this state. Confirm every value with the issuing authority before relying on it.'
FROM (
    VALUES ('Front', 12.00), ('Rear', 14.00), ('Police', 16.00)) AS v("role", "threshold")
WHERE
    EXISTS (SELECT 1 FROM "jurisdiction_rules" r WHERE r."id" = 'jr_seed_WY')
ON CONFLICT
    DO NOTHING;