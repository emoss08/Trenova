-- # Copyright 2023-2025 Eric Moss
-- # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

CREATE TYPE model_type_enum AS ENUM (
    'DocumentQuality'
);

CREATE TYPE model_status_enum AS ENUM (
    'Stable', -- Model is stable and ready for production
    'Beta', -- Model is in beta testing
    'Legacy' -- Model is deprecated and no longer in use
);

CREATE TYPE feedback_type_enum AS ENUM (
    'Good', -- Document was correctly assessed
    'Bad', -- Document was not correctly assessed
    'Unclear' -- Document was not clear enough to assess
);

-- bun:split
CREATE TABLE IF NOT EXISTS "pretrained_models" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "name" varchar(100) NOT NULL,
    "version" varchar(50) NOT NULL,
    "type" model_type_enum NOT NULL,
    -- Model Details
    "description" text NOT NULL,
    "status" model_status_enum NOT NULL DEFAULT 'Stable',
    "path" varchar(255) NOT NULL,
    "is_default" boolean NOT NULL DEFAULT FALSE,
    "is_active" boolean NOT NULL DEFAULT TRUE,
    -- Training Info
    "trained_at" bigint NOT NULL DEFAULT 0,
    "training_metrics" jsonb NOT NULL DEFAULT '{}' ::jsonb,
    -- Metadata
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_pretrained_models" PRIMARY KEY ("id"),
    CONSTRAINT "check_pretrained_training_metrics_format" CHECK (jsonb_typeof(training_metrics) = 'object')
);

-- bun:split
-- Indexes for pretrained_models table
CREATE INDEX "idx_pretrained_models_type" ON "pretrained_models" ("type");

CREATE INDEX "idx_pretrained_models_status" ON "pretrained_models" ("status");

CREATE INDEX "idx_pretrained_models_created_updated" ON "pretrained_models" ("created_at", "updated_at");

COMMENT ON TABLE "pretrained_models" IS 'Stores information about pretrained models';

-- bun:split
-- Document Quality Configs
CREATE TABLE IF NOT EXISTS "document_quality_configs" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    -- Core fields
    "is_active" boolean NOT NULL DEFAULT TRUE,
    -- Quality Thresholds
    "min_dpi" int NOT NULL DEFAULT 200,
    "min_brightness" numeric(5, 2) NOT NULL DEFAULT 40,
    "max_brightness" numeric(5, 2) NOT NULL DEFAULT 220,
    "min_contrast" numeric(5, 2) NOT NULL DEFAULT 40,
    "min_sharpness" numeric(5, 2) NOT NULL DEFAULT 50,
    -- OCR Configuration
    "min_word_count" int NOT NULL DEFAULT 50,
    "min_text_density" numeric(5, 2) NOT NULL DEFAULT 0.1,
    -- Model Settings
    "model_id" varchar(100) NOT NULL,
    "allow_training" boolean NOT NULL DEFAULT TRUE,
    "auto_reject_score" numeric(3, 2) NOT NULL DEFAULT 0.2,
    "manual_review_score" numeric(3, 2) NOT NULL DEFAULT 0.4,
    "min_confidence" numeric(3, 2) NOT NULL DEFAULT 0.7,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_document_quality_configs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_document_quality_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_configs_model" FOREIGN KEY ("model_id") REFERENCES "pretrained_models" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "unique_document_quality_organization_id" UNIQUE ("organization_id") -- Ensures only one document quality config per organization
);

--bun:split
-- indexes for document_quality_configs table
CREATE INDEX "idx_document_quality_configs_business_unit" ON "document_quality_configs" ("business_unit_id");

CREATE INDEX "idx_document_quality_configs_organization" ON "document_quality_configs" ("organization_id");

CREATE INDEX "idx_document_quality_configs_model" ON "document_quality_configs" ("model_id");

CREATE INDEX "idx_document_quality_configs_created_updated" ON "document_quality_configs" ("created_at", "updated_at");

COMMENT ON TABLE "document_quality_configs" IS 'Stores information about document quality configs';

--bun:split
-- Document Quality Feedback
CREATE TABLE IF NOT EXISTS "document_quality_feedback" (
    -- Primary identifiers
    "id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "user_id" varchar(100) NOT NULL,
    -- Feedback Details
    "document_url" text NOT NULL,
    "feedback_type" feedback_type_enum NOT NULL,
    "comment" text,
    -- Quality Metrics
    "quality_score" numeric(5, 2) NOT NULL,
    "confidence_score" numeric(5, 2) NOT NULL,
    "sharpness" numeric(10, 2) NOT NULL,
    "text_density" numeric(5, 2) NOT NULL,
    "word_count" int NOT NULL,
    -- Training Flags
    "used_for_training" boolean NOT NULL DEFAULT FALSE,
    "trained_at" bigint,
    -- Metadata
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    "updated_at" bigint NOT NULL DEFAULT EXTRACT(EPOCH FROM current_timestamp) ::bigint,
    -- Constraints
    CONSTRAINT "pk_document_quality_feedback" PRIMARY KEY ("id", "organization_id", "business_unit_id", "user_id"),
    CONSTRAINT "fk_document_quality_feedback_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_feedback_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_feedback_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- indexes for document_quality_feedback table
CREATE INDEX "idx_document_quality_feedback_organization" ON "document_quality_feedback" ("organization_id");

CREATE INDEX "idx_document_quality_feedback_business_unit" ON "document_quality_feedback" ("business_unit_id");

CREATE INDEX "idx_document_quality_feedback_user" ON "document_quality_feedback" ("user_id");

CREATE INDEX "idx_document_quality_feedback_created_updated" ON "document_quality_feedback" ("created_at", "updated_at");

COMMENT ON TABLE "document_quality_feedback" IS 'Stores information about document quality feedback';

