-- How an organization shadows production document extraction with a candidate
-- AI provider: which provider, what share of extractions, and how many a day.
CREATE TABLE IF NOT EXISTS "extraction_shadow_settings" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "enabled" boolean NOT NULL DEFAULT FALSE,
    "provider_id" varchar(100),
    "sample_percent" integer NOT NULL DEFAULT 10,
    "daily_limit" integer NOT NULL DEFAULT 200,
    "updated_by_id" varchar(100),
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_shadow_settings" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_shadow_settings_tenant" UNIQUE ("organization_id", "business_unit_id"),
    CONSTRAINT "ck_extraction_shadow_settings_sample_percent" CHECK ("sample_percent" BETWEEN 1 AND 100),
    CONSTRAINT "ck_extraction_shadow_settings_daily_limit" CHECK ("daily_limit" BETWEEN 1 AND 5000),
    CONSTRAINT "ck_extraction_shadow_settings_provider" CHECK (NOT "enabled" OR "provider_id" IS NOT NULL),
    CONSTRAINT "fk_extraction_shadow_settings_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_shadow_settings_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
-- One production extraction run again on the candidate provider. Its answer is
-- kept, never applied, and scored against the shipment a person confirmed
-- beside production's own answer for the same document.
CREATE TABLE IF NOT EXISTS "extraction_shadow_results" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "document_id" varchar(100) NOT NULL,
    "extracted_at" bigint NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "status_reason" text,
    "provider_id" varchar(100) NOT NULL,
    "provider_name" varchar(100) NOT NULL,
    "served_model" varchar(255),
    "production_provider_id" varchar(100),
    "production_model" varchar(255),
    "accepted" boolean NOT NULL DEFAULT FALSE,
    "rejection_reason" text,
    "draft_data" jsonb,
    "predicted" jsonb,
    "correction_id" varchar(100),
    "scored_at" bigint,
    "verdict" varchar(20),
    "field_results" jsonb NOT NULL DEFAULT '[]',
    "scored_count" integer NOT NULL DEFAULT 0,
    "correct_count" integer NOT NULL DEFAULT 0,
    "corrected_count" integer NOT NULL DEFAULT 0,
    "missed_count" integer NOT NULL DEFAULT 0,
    "accuracy" double precision NOT NULL DEFAULT 0,
    "baseline_field_results" jsonb NOT NULL DEFAULT '[]',
    "baseline_scored_count" integer NOT NULL DEFAULT 0,
    "baseline_correct_count" integer NOT NULL DEFAULT 0,
    "baseline_corrected_count" integer NOT NULL DEFAULT 0,
    "baseline_missed_count" integer NOT NULL DEFAULT 0,
    "baseline_accuracy" double precision NOT NULL DEFAULT 0,
    "latency_ms" bigint NOT NULL DEFAULT 0,
    "input_tokens" bigint NOT NULL DEFAULT 0,
    "output_tokens" bigint NOT NULL DEFAULT 0,
    "cost_usd" numeric(14,6) NOT NULL DEFAULT 0,
    "workflow_id" varchar(255),
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_shadow_results" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_shadow_results_extraction" UNIQUE ("organization_id", "business_unit_id", "document_id", "extracted_at"),
    CONSTRAINT "ck_extraction_shadow_results_status" CHECK ("status" IN ('Pending', 'Completed', 'Failed', 'Skipped')),
    CONSTRAINT "ck_extraction_shadow_results_verdict" CHECK ("verdict" IS NULL OR "verdict" IN ('Better', 'Worse', 'Same')),
    CONSTRAINT "ck_extraction_shadow_results_scored" CHECK ("scored_count" = "correct_count" + "corrected_count" + "missed_count"),
    CONSTRAINT "ck_extraction_shadow_results_baseline_scored" CHECK ("baseline_scored_count" = "baseline_correct_count" + "baseline_corrected_count" + "baseline_missed_count"),
    CONSTRAINT "ck_extraction_shadow_results_accuracy" CHECK ("accuracy" >= 0 AND "accuracy" <= 1 AND "baseline_accuracy" >= 0 AND "baseline_accuracy" <= 1),
    CONSTRAINT "ck_extraction_shadow_results_score" CHECK (("correction_id" IS NULL) = ("scored_at" IS NULL) AND ("correction_id" IS NULL) = ("verdict" IS NULL)),
    CONSTRAINT "fk_extraction_shadow_results_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_shadow_results_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_shadow_results_created"
    ON "extraction_shadow_results" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_shadow_results_document"
    ON "extraction_shadow_results" ("organization_id", "business_unit_id", "document_id");

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_shadow_results_scored"
    ON "extraction_shadow_results" ("organization_id", "business_unit_id", "provider_id", "scored_at" DESC)
    WHERE "scored_at" IS NOT NULL;

--bun:split
-- A shadow extraction finishing after its document was confirmed looks the
-- correction up by document.
CREATE INDEX IF NOT EXISTS "idx_ai_corrections_document"
    ON "ai_corrections" ("organization_id", "business_unit_id", "document_id", "captured_at" DESC)
    WHERE "document_id" IS NOT NULL;

COMMENT ON TABLE "extraction_shadow_settings" IS 'Which AI provider shadows production document extraction for an organization, on what share of extractions and how many a day';

COMMENT ON COLUMN "extraction_shadow_settings"."sample_percent" IS 'The share of production extractions also sent to the candidate, chosen by a hash of the document and extraction time';

COMMENT ON COLUMN "extraction_shadow_settings"."daily_limit" IS 'The most shadow extractions started in any 24 hours';

COMMENT ON TABLE "extraction_shadow_results" IS 'A production extraction run again on a candidate AI provider; kept, never applied, and scored against the confirmed shipment beside production';

COMMENT ON COLUMN "extraction_shadow_results"."draft_data" IS 'The shipment draft the candidate would have produced, merged with the same rule-based reading production was merged with';

COMMENT ON COLUMN "extraction_shadow_results"."baseline_field_results" IS 'How production''s draft for the same document scored against the same confirmed shipment';

COMMENT ON COLUMN "extraction_shadow_results"."verdict" IS 'Whether the candidate did better, worse or the same as production on this document';
