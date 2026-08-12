-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20250105193652_document_quality.tx.up.sql

CREATE TABLE IF NOT EXISTS "pretrained_models" (
    "id" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "version" TEXT NOT NULL,
    "type" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Stable',
    "path" TEXT NOT NULL,
    "is_default" INTEGER NOT NULL DEFAULT 0,
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "trained_at" INTEGER NOT NULL DEFAULT 0,
    "training_metrics" TEXT NOT NULL DEFAULT '{}',
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_pretrained_models" PRIMARY KEY ("id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pretrained_models_type" ON "pretrained_models" ("type");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pretrained_models_status" ON "pretrained_models" ("status");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_pretrained_models_created_updated" ON "pretrained_models" ("created_at", "updated_at");

--bun:split

CREATE TABLE IF NOT EXISTS "document_quality_configs" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "is_active" INTEGER NOT NULL DEFAULT 1,
    "min_dpi" INTEGER NOT NULL DEFAULT 200,
    "min_brightness" REAL NOT NULL DEFAULT 40,
    "max_brightness" REAL NOT NULL DEFAULT 220,
    "min_contrast" REAL NOT NULL DEFAULT 40,
    "min_sharpness" REAL NOT NULL DEFAULT 50,
    "min_word_count" INTEGER NOT NULL DEFAULT 50,
    "min_text_density" REAL NOT NULL DEFAULT 0.1,
    "model_id" TEXT NOT NULL,
    "allow_training" INTEGER NOT NULL DEFAULT 1,
    "auto_reject_score" REAL NOT NULL DEFAULT 0.2,
    "manual_review_score" REAL NOT NULL DEFAULT 0.4,
    "min_confidence" REAL NOT NULL DEFAULT 0.7,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_quality_configs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "fk_document_quality_configs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_configs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_configs_model" FOREIGN KEY ("model_id") REFERENCES "pretrained_models" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "unique_document_quality_organization_id" UNIQUE ("organization_id")
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_configs_business_unit" ON "document_quality_configs" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_configs_organization" ON "document_quality_configs" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_configs_model" ON "document_quality_configs" ("model_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_configs_created_updated" ON "document_quality_configs" ("created_at", "updated_at");

--bun:split

CREATE TABLE IF NOT EXISTS "document_quality_feedback" (
    "id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "user_id" TEXT NOT NULL,
    "document_url" TEXT NOT NULL,
    "feedback_type" TEXT NOT NULL,
    "comment" TEXT,
    "quality_score" REAL NOT NULL,
    "confidence_score" REAL NOT NULL,
    "sharpness" REAL NOT NULL,
    "text_density" REAL NOT NULL,
    "word_count" INTEGER NOT NULL,
    "used_for_training" INTEGER NOT NULL DEFAULT 0,
    "trained_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_document_quality_feedback" PRIMARY KEY ("id", "organization_id", "business_unit_id", "user_id"),
    CONSTRAINT "fk_document_quality_feedback_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_feedback_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_document_quality_feedback_user" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_feedback_organization" ON "document_quality_feedback" ("organization_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_feedback_business_unit" ON "document_quality_feedback" ("business_unit_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_feedback_user" ON "document_quality_feedback" ("user_id");

--bun:split

CREATE INDEX IF NOT EXISTS "idx_document_quality_feedback_created_updated" ON "document_quality_feedback" ("created_at", "updated_at");
