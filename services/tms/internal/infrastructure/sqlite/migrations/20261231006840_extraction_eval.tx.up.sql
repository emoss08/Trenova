-- Code generated from the PostgreSQL migrations by
-- scripts/dialect-convert/convert.py. Hand-edits are preserved only if you
-- stop regenerating this file; see docs/databases.md.
-- Source: 20261231006840_extraction_eval.tx.up.sql

CREATE TABLE IF NOT EXISTS "extraction_eval_cases" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "task" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Candidate',
    "title" TEXT NOT NULL,
    "document_kind" TEXT,
    "document_fingerprint" TEXT,
    "file_name" TEXT,
    "pages" TEXT NOT NULL,
    "page_count" INTEGER NOT NULL DEFAULT 0,
    "input_hash" TEXT NOT NULL,
    "expected" TEXT NOT NULL,
    "expected_field_count" INTEGER NOT NULL DEFAULT 0,
    "source_correction_id" TEXT,
    "source_document_id" TEXT,
    "notes" TEXT,
    "created_by_id" TEXT NOT NULL,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    CONSTRAINT "pk_extraction_eval_cases" PRIMARY KEY ("id", "business_unit_id", "organization_id"),
    CONSTRAINT "uq_extraction_eval_cases_input" UNIQUE ("organization_id", "business_unit_id", "task", "input_hash"),
    CONSTRAINT "ck_extraction_eval_cases_task" CHECK ("task" IN ('ShipmentDraftExtraction')),
    CONSTRAINT "ck_extraction_eval_cases_status" CHECK ("status" IN ('Candidate', 'Active', 'Retired')),
    CONSTRAINT "ck_extraction_eval_cases_input_hash" CHECK (length("input_hash") = 64),
    CONSTRAINT "ck_extraction_eval_cases_notes_length" CHECK ("notes" IS NULL OR length("notes") <= 2000),
    CONSTRAINT "fk_extraction_eval_cases_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_extraction_eval_cases_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

--bun:split

CREATE INDEX IF NOT EXISTS "idx_extraction_eval_cases_status"
    ON "extraction_eval_cases" ("organization_id", "business_unit_id", "task", "status", "created_at" DESC);

--bun:split

CREATE UNIQUE INDEX IF NOT EXISTS "uq_extraction_eval_cases_source_correction"
    ON "extraction_eval_cases" ("organization_id", "business_unit_id", "source_correction_id")WHERE "source_correction_id" IS NOT NULL;

--bun:split

CREATE TABLE IF NOT EXISTS "extraction_eval_runs" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "task" TEXT NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Queued',
    "provider_id" TEXT NOT NULL,
    "provider_name" TEXT NOT NULL,
    "provider_model" TEXT,
    "served_model" TEXT,
    "case_limit" INTEGER NOT NULL,
    "cases_total" INTEGER NOT NULL DEFAULT 0,
    "cases_completed" INTEGER NOT NULL DEFAULT 0,
    "cases_failed" INTEGER NOT NULL DEFAULT 0,
    "cases_skipped" INTEGER NOT NULL DEFAULT 0,
    "scored_count" INTEGER NOT NULL DEFAULT 0,
    "correct_count" INTEGER NOT NULL DEFAULT 0,
    "corrected_count" INTEGER NOT NULL DEFAULT 0,
    "missed_count" INTEGER NOT NULL DEFAULT 0,
    "unconfirmed_count" INTEGER NOT NULL DEFAULT 0,
    "unscored_count" INTEGER NOT NULL DEFAULT 0,
    "accuracy" REAL NOT NULL DEFAULT 0,
    "field_accuracy" TEXT NOT NULL DEFAULT '[]',
    "cost_usd" REAL NOT NULL DEFAULT 0,
    "input_tokens" INTEGER NOT NULL DEFAULT 0,
    "output_tokens" INTEGER NOT NULL DEFAULT 0,
    "avg_latency_ms" INTEGER NOT NULL DEFAULT 0,
    "stop_reason" TEXT,
    "failure_message" TEXT,
    "requested_by_id" TEXT NOT NULL,
    "workflow_id" TEXT,
    "started_at" INTEGER,
    "finished_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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

CREATE UNIQUE INDEX IF NOT EXISTS "uq_extraction_eval_runs_active"
    ON "extraction_eval_runs" ("organization_id", "business_unit_id", "task")WHERE "status" IN ('Queued', 'Running');

--bun:split

CREATE TABLE IF NOT EXISTS "extraction_eval_results" (
    "id" TEXT NOT NULL,
    "business_unit_id" TEXT NOT NULL,
    "organization_id" TEXT NOT NULL,
    "run_id" TEXT NOT NULL,
    "case_id" TEXT NOT NULL,
    "case_title" TEXT NOT NULL,
    "ordinal" INTEGER NOT NULL,
    "status" TEXT NOT NULL DEFAULT 'Pending',
    "model" TEXT,
    "provider_id" TEXT,
    "predicted" TEXT,
    "field_results" TEXT NOT NULL DEFAULT '[]',
    "scored_count" INTEGER NOT NULL DEFAULT 0,
    "correct_count" INTEGER NOT NULL DEFAULT 0,
    "corrected_count" INTEGER NOT NULL DEFAULT 0,
    "missed_count" INTEGER NOT NULL DEFAULT 0,
    "unconfirmed_count" INTEGER NOT NULL DEFAULT 0,
    "unscored_count" INTEGER NOT NULL DEFAULT 0,
    "accuracy" REAL NOT NULL DEFAULT 0,
    "latency_ms" INTEGER NOT NULL DEFAULT 0,
    "input_tokens" INTEGER NOT NULL DEFAULT 0,
    "output_tokens" INTEGER NOT NULL DEFAULT 0,
    "cost_usd" REAL NOT NULL DEFAULT 0,
    "error_message" TEXT,
    "started_at" INTEGER,
    "completed_at" INTEGER,
    "version" INTEGER NOT NULL DEFAULT 0,
    "created_at" INTEGER NOT NULL DEFAULT (unixepoch()),
    "updated_at" INTEGER NOT NULL DEFAULT (unixepoch()),
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
