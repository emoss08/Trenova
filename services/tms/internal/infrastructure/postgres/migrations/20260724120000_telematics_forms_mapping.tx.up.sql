ALTER TABLE "dispatch_controls"
    ADD COLUMN IF NOT EXISTS "enable_auto_stop_actuals" boolean NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS "telematics_form_submissions"(
    "id" character varying(100) NOT NULL,
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "provider" character varying(32) NOT NULL DEFAULT 'Samsara',
    "provider_submission_id" text NOT NULL,
    "template_id" text NOT NULL,
    "template_name" text,
    "worker_id" character varying(100),
    "shipment_id" character varying(100),
    "shipment_move_id" character varying(100),
    "stop_id" character varying(100),
    "submitted_at" bigint NOT NULL,
    "fields" jsonb,
    "applied" boolean NOT NULL DEFAULT false,
    "applied_fields" integer NOT NULL DEFAULT 0,
    "applied_at" bigint,
    "created_at" bigint NOT NULL,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_form_submissions_provider_id
    ON "telematics_form_submissions" ("organization_id", "business_unit_id", "provider", "provider_submission_id");

CREATE INDEX IF NOT EXISTS idx_telematics_form_submissions_shipment
    ON "telematics_form_submissions" ("organization_id", "business_unit_id", "shipment_id", "submitted_at" DESC);

CREATE INDEX IF NOT EXISTS idx_telematics_form_submissions_submitted_at
    ON "telematics_form_submissions" ("organization_id", "business_unit_id", "submitted_at" DESC);

CREATE TABLE IF NOT EXISTS "telematics_form_mappings"(
    "id" character varying(100) NOT NULL,
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "provider" character varying(32) NOT NULL DEFAULT 'Samsara',
    "template_id" text NOT NULL,
    "template_name" text,
    "name" character varying(200) NOT NULL,
    "description" text,
    "enabled" boolean NOT NULL DEFAULT true,
    "version" bigint,
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_telematics_form_mappings_provider_template
    ON "telematics_form_mappings" ("organization_id", "business_unit_id", "provider", "template_id");

CREATE TABLE IF NOT EXISTS "telematics_form_mapping_items"(
    "id" character varying(100) NOT NULL,
    "organization_id" character varying(100) NOT NULL,
    "business_unit_id" character varying(100) NOT NULL,
    "mapping_id" character varying(100) NOT NULL,
    "source_field_label" text NOT NULL,
    "target_kind" character varying(32) NOT NULL,
    "target_field" character varying(64),
    "target_custom_field_key" character varying(100),
    "created_at" bigint NOT NULL DEFAULT extract(epoch FROM current_timestamp)::bigint,
    PRIMARY KEY ("id", "organization_id", "business_unit_id")
);

CREATE INDEX IF NOT EXISTS idx_telematics_form_mapping_items_mapping
    ON "telematics_form_mapping_items" ("organization_id", "business_unit_id", "mapping_id");
