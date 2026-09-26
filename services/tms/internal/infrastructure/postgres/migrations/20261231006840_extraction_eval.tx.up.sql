-- A frozen document extraction test: the page text the extractor reads and the
-- values a person confirmed from it. Promoted from an AI correction, so the case
-- still runs after the document is re-extracted or deleted.
CREATE TABLE IF NOT EXISTS "extraction_eval_cases" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "task" varchar(50) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Candidate',
    "title" varchar(200) NOT NULL,
    "document_kind" varchar(100),
    "document_fingerprint" varchar(255),
    "file_name" varchar(255),
    "pages" jsonb NOT NULL,
    "page_count" integer NOT NULL DEFAULT 0,
    "input_hash" varchar(64) NOT NULL,
    "expected" jsonb NOT NULL,
    "expected_field_count" integer NOT NULL DEFAULT 0,
    "source_correction_id" varchar(100),
    "source_document_id" varchar(100),
    "notes" text,
    "created_by_id" varchar(100) NOT NULL,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_eval_cases" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_eval_cases_input" UNIQUE ("organization_id", "business_unit_id", "task", "input_hash"),
    CONSTRAINT "ck_extraction_eval_cases_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    CONSTRAINT "ck_extraction_eval_cases_status" CHECK ("status" IN ('Candidate', 'Active', 'Retired')),
    CONSTRAINT "ck_extraction_eval_cases_input_hash" CHECK (char_length("input_hash") = 64),
    CONSTRAINT "ck_extraction_eval_cases_notes_length" CHECK ("notes" IS NULL OR char_length("notes") <= 2000),
    CONSTRAINT "fk_extraction_eval_cases_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_cases_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_eval_cases_status"
    ON "extraction_eval_cases" ("organization_id", "business_unit_id", "task", "status", "created_at" DESC);

--bun:split
CREATE UNIQUE INDEX IF NOT EXISTS "uq_extraction_eval_cases_source_correction"
    ON "extraction_eval_cases" ("organization_id", "business_unit_id", "source_correction_id")
    WHERE "source_correction_id" IS NOT NULL;

--bun:split
-- One model, pinned to one provider, run over the active cases.
CREATE TABLE IF NOT EXISTS "extraction_eval_runs" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "task" varchar(50) NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Queued',
    "provider_id" varchar(100) NOT NULL,
    "provider_name" varchar(100) NOT NULL,
    "provider_model" varchar(255),
    "served_model" varchar(255),
    "case_limit" integer NOT NULL,
    "cases_total" integer NOT NULL DEFAULT 0,
    "cases_completed" integer NOT NULL DEFAULT 0,
    "cases_failed" integer NOT NULL DEFAULT 0,
    "cases_skipped" integer NOT NULL DEFAULT 0,
    "scored_count" integer NOT NULL DEFAULT 0,
    "correct_count" integer NOT NULL DEFAULT 0,
    "corrected_count" integer NOT NULL DEFAULT 0,
    "missed_count" integer NOT NULL DEFAULT 0,
    "unconfirmed_count" integer NOT NULL DEFAULT 0,
    "unscored_count" integer NOT NULL DEFAULT 0,
    "accuracy" double precision NOT NULL DEFAULT 0,
    "field_accuracy" jsonb NOT NULL DEFAULT '[]',
    "cost_usd" numeric(14,6) NOT NULL DEFAULT 0,
    "input_tokens" bigint NOT NULL DEFAULT 0,
    "output_tokens" bigint NOT NULL DEFAULT 0,
    "avg_latency_ms" bigint NOT NULL DEFAULT 0,
    "stop_reason" text,
    "failure_message" text,
    "requested_by_id" varchar(100) NOT NULL,
    "workflow_id" varchar(255),
    "started_at" bigint,
    "finished_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_eval_runs" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "ck_extraction_eval_runs_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    CONSTRAINT "ck_extraction_eval_runs_status" CHECK ("status" IN ('Queued', 'Running', 'Completed', 'BudgetStopped', 'Canceled', 'Failed')),
    CONSTRAINT "ck_extraction_eval_runs_case_limit" CHECK ("case_limit" BETWEEN 1 AND 500),
    CONSTRAINT "ck_extraction_eval_runs_accuracy" CHECK ("accuracy" >= 0 AND "accuracy" <= 1),
    CONSTRAINT "fk_extraction_eval_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_eval_runs_created"
    ON "extraction_eval_runs" ("organization_id", "business_unit_id", "created_at" DESC);

--bun:split
-- Only one run of a task is in flight per organization at a time.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_extraction_eval_runs_active"
    ON "extraction_eval_runs" ("organization_id", "business_unit_id", "task")
    WHERE "status" IN ('Queued', 'Running');

--bun:split
-- One case's outcome within a run.
CREATE TABLE IF NOT EXISTS "extraction_eval_results" (
    "id" varchar(100) NOT NULL,
    "business_unit_id" varchar(100) NOT NULL,
    "organization_id" varchar(100) NOT NULL,
    "run_id" varchar(100) NOT NULL,
    "case_id" varchar(100) NOT NULL,
    "case_title" varchar(200) NOT NULL,
    "ordinal" integer NOT NULL,
    "status" varchar(20) NOT NULL DEFAULT 'Pending',
    "model" varchar(255),
    "provider_id" varchar(100),
    "predicted" jsonb,
    "field_results" jsonb NOT NULL DEFAULT '[]',
    "scored_count" integer NOT NULL DEFAULT 0,
    "correct_count" integer NOT NULL DEFAULT 0,
    "corrected_count" integer NOT NULL DEFAULT 0,
    "missed_count" integer NOT NULL DEFAULT 0,
    "unconfirmed_count" integer NOT NULL DEFAULT 0,
    "unscored_count" integer NOT NULL DEFAULT 0,
    "accuracy" double precision NOT NULL DEFAULT 0,
    "latency_ms" bigint NOT NULL DEFAULT 0,
    "input_tokens" bigint NOT NULL DEFAULT 0,
    "output_tokens" bigint NOT NULL DEFAULT 0,
    "cost_usd" numeric(14,6) NOT NULL DEFAULT 0,
    "error_message" text,
    "started_at" bigint,
    "completed_at" bigint,
    "version" bigint NOT NULL DEFAULT 0,
    "created_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    "updated_at" bigint NOT NULL DEFAULT extract(epoch from current_timestamp)::bigint,
    CONSTRAINT "pk_extraction_eval_results" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_eval_results_run_case" UNIQUE ("organization_id", "business_unit_id", "run_id", "case_id"),
    CONSTRAINT "uq_extraction_eval_results_run_ordinal" UNIQUE ("organization_id", "business_unit_id", "run_id", "ordinal"),
    CONSTRAINT "ck_extraction_eval_results_status" CHECK ("status" IN ('Pending', 'Completed', 'Failed', 'Skipped')),
    CONSTRAINT "ck_extraction_eval_results_scored" CHECK ("scored_count" = "correct_count" + "corrected_count" + "missed_count"),
    CONSTRAINT "fk_extraction_eval_results_run" FOREIGN KEY ("run_id", "business_unit_id", "organization_id") REFERENCES "extraction_eval_runs"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_results_case" FOREIGN KEY ("case_id", "business_unit_id", "organization_id") REFERENCES "extraction_eval_cases"("id", "business_unit_id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_results_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_results_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split
CREATE INDEX IF NOT EXISTS "idx_extraction_eval_results_case"
    ON "extraction_eval_results" ("organization_id", "business_unit_id", "case_id");

COMMENT ON TABLE "extraction_eval_cases" IS 'A frozen extraction test: the page text the extractor reads and the values a person confirmed, promoted from an AI correction';

COMMENT ON COLUMN "extraction_eval_cases"."status" IS 'Candidate until a person accepts it, Active while it is run, Retired once it is no longer run';

COMMENT ON COLUMN "extraction_eval_cases"."pages" IS 'The page text sent to the model, frozen when the case was created';

COMMENT ON COLUMN "extraction_eval_cases"."input_hash" IS 'SHA-256 of the task, file name and pages, so the same input is not added twice';

COMMENT ON COLUMN "extraction_eval_cases"."expected" IS 'The confirmed field values and stops a run is scored against';

COMMENT ON COLUMN "extraction_eval_cases"."source_correction_id" IS 'The AI correction the case was promoted from';

COMMENT ON TABLE "extraction_eval_runs" IS 'One model, pinned to one AI provider, run over the active extraction cases and scored field by field';

COMMENT ON COLUMN "extraction_eval_runs"."provider_model" IS 'The model the provider was configured with when the run was requested';

COMMENT ON COLUMN "extraction_eval_runs"."served_model" IS 'The model the provider reported serving the calls';

COMMENT ON COLUMN "extraction_eval_runs"."accuracy" IS 'Correct fields divided by scored fields (correct, corrected and missed)';

COMMENT ON COLUMN "extraction_eval_runs"."field_accuracy" IS 'Accuracy per field, with stop fields grouped across stops, worst first';

COMMENT ON COLUMN "extraction_eval_runs"."stop_reason" IS 'Why the run ended before every case ran: the evaluation budget, or a person canceling it';

COMMENT ON TABLE "extraction_eval_results" IS 'One extraction case evaluated within a run: what the model predicted and how each field scored';
