ALTER TABLE "agent_extensions"
    DROP CONSTRAINT IF EXISTS "ck_agent_extensions_type";

ALTER TABLE "agent_extensions"
    ADD CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa', 'Parallel', 'BrowserUse', 'E2B'));

CREATE TABLE IF NOT EXISTS "agent_analysis_runs" (
    "id" VARCHAR(100) NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "extension_type" VARCHAR(50) NOT NULL,
    "status" VARCHAR(20) NOT NULL,
    "purpose" VARCHAR(500) NOT NULL DEFAULT '',
    "code" TEXT NOT NULL,
    "datasets" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "stdout" TEXT NOT NULL DEFAULT '',
    "stderr" TEXT NOT NULL DEFAULT '',
    "results" JSONB NOT NULL DEFAULT '[]'::jsonb,
    "error_name" VARCHAR(200) NOT NULL DEFAULT '',
    "error_value" TEXT NOT NULL DEFAULT '',
    "traceback" TEXT NOT NULL DEFAULT '',
    "chart_count" INTEGER NOT NULL DEFAULT 0,
    "failure" TEXT NOT NULL DEFAULT '',
    "vendor_sandbox_id" VARCHAR(100) NOT NULL DEFAULT '',
    "duration_ms" BIGINT NOT NULL DEFAULT 0,
    "cost_usd" NUMERIC(19, 6) NOT NULL DEFAULT 0,
    "agent_definition_id" VARCHAR(100),
    "agent_run_id" VARCHAR(100),
    "thread_id" VARCHAR(100),
    "requested_by_id" VARCHAR(100),
    "started_at" BIGINT NOT NULL,
    "finished_at" BIGINT,
    "version" BIGINT NOT NULL DEFAULT 0,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    "updated_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_analysis_runs" PRIMARY KEY ("id", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_analysis_runs_business_unit" FOREIGN KEY ("business_unit_id") REFERENCES "business_units" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_analysis_runs_organization" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "fk_agent_analysis_runs_requested_by" FOREIGN KEY ("requested_by_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
    CONSTRAINT "ck_agent_analysis_runs_status" CHECK ("status" IN ('Running', 'Succeeded', 'Errored', 'Failed', 'TimedOut')),
    CONSTRAINT "ck_agent_analysis_runs_datasets" CHECK (jsonb_typeof("datasets") = 'array'),
    CONSTRAINT "ck_agent_analysis_runs_results" CHECK (jsonb_typeof("results") = 'array'),
    CONSTRAINT "ck_agent_analysis_runs_counts" CHECK ("cost_usd" >= 0 AND "duration_ms" >= 0 AND "chart_count" BETWEEN 0 AND 4)
);

CREATE INDEX IF NOT EXISTS "idx_agent_analysis_runs_tenant_started" ON "agent_analysis_runs" ("organization_id", "business_unit_id", "started_at" DESC);

COMMENT ON TABLE "agent_analysis_runs" IS 'Every analysis an agent ran in a data analysis sandbox: what it was for, the code, the datasets it read and where they came from, what it printed and raised, and its cost';

COMMENT ON COLUMN "agent_analysis_runs"."datasets" IS 'The report runs and attached files the analysis read, each with the path the code opened it at; their rows are not kept here';

CREATE TABLE IF NOT EXISTS "agent_analysis_charts" (
    "run_id" VARCHAR(100) NOT NULL,
    "position" INTEGER NOT NULL,
    "business_unit_id" VARCHAR(100) NOT NULL,
    "organization_id" VARCHAR(100) NOT NULL,
    "title" VARCHAR(200) NOT NULL DEFAULT '',
    "image" BYTEA NOT NULL,
    "created_at" BIGINT NOT NULL DEFAULT EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::BIGINT,
    CONSTRAINT "pk_agent_analysis_charts" PRIMARY KEY ("run_id", "position", "organization_id", "business_unit_id"),
    CONSTRAINT "fk_agent_analysis_charts_run" FOREIGN KEY ("run_id", "organization_id", "business_unit_id") REFERENCES "agent_analysis_runs" ("id", "organization_id", "business_unit_id") ON UPDATE NO ACTION ON DELETE CASCADE,
    CONSTRAINT "ck_agent_analysis_charts_position" CHECK ("position" BETWEEN 1 AND 4),
    CONSTRAINT "ck_agent_analysis_charts_size" CHECK (octet_length("image") <= 2097152)
);

COMMENT ON TABLE "agent_analysis_charts" IS 'The images an analysis displayed, in the order it showed them, kept with the run so the person sees them beside the reply';

ALTER TABLE "assistant_artifacts"
    DROP CONSTRAINT IF EXISTS "ck_assistant_artifacts_kind";

ALTER TABLE "assistant_artifacts"
    ADD CONSTRAINT "ck_assistant_artifacts_kind" CHECK (
        "kind" IN (
            'report_preview', 'report_run', 'email_draft', 'plan', 'entity_card',
            'table_view', 'rate_explanation', 'dashboard_ref', 'briefing',
            'inbound_message', 'run_diff', 'document', 'navigation', 'draft_edit',
            'decision_request', 'extraction', 'edi_connection_draft', 'analysis'
        )
    );

SELECT trenova_rls.reconcile();
