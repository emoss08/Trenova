-- Data analysis moved off E2B onto Trenova's own script interpreter, and its tool,
-- report_analyze_data, into agent_run_script. A script that shows the person a figure,
-- table or chart is kept as an analysis run, its charts as specs the client draws rather
-- than images. The runs E2B made point at sandboxes, costs and chart images that no
-- longer exist, so they go with it, and so does E2B's extension.
DROP TABLE IF EXISTS "agent_analysis_charts";

DELETE FROM "agent_analysis_runs";

ALTER TABLE "agent_analysis_runs"
    DROP CONSTRAINT IF EXISTS "ck_agent_analysis_runs_status",
    DROP CONSTRAINT IF EXISTS "ck_agent_analysis_runs_results",
    DROP CONSTRAINT IF EXISTS "ck_agent_analysis_runs_counts",
    DROP COLUMN IF EXISTS "extension_type",
    DROP COLUMN IF EXISTS "stderr",
    DROP COLUMN IF EXISTS "results",
    DROP COLUMN IF EXISTS "chart_count",
    DROP COLUMN IF EXISTS "vendor_sandbox_id",
    DROP COLUMN IF EXISTS "cost_usd",
    DROP COLUMN IF EXISTS "version",
    DROP COLUMN IF EXISTS "updated_at",
    ADD COLUMN IF NOT EXISTS "charts" JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE "agent_analysis_runs" RENAME COLUMN "stdout" TO "output";

ALTER TABLE "agent_analysis_runs"
    ALTER COLUMN "finished_at" SET NOT NULL,
    ADD CONSTRAINT "ck_agent_analysis_runs_status" CHECK ("status" IN ('Succeeded', 'Errored', 'Stopped', 'TimedOut', 'Failed')),
    ADD CONSTRAINT "ck_agent_analysis_runs_shown" CHECK (
        jsonb_typeof("tables") = 'array'
        AND jsonb_typeof("figures") = 'array'
        AND jsonb_typeof("charts") = 'array'
        AND jsonb_typeof("layout") = 'array'
    ),
    ADD CONSTRAINT "ck_agent_analysis_runs_timing" CHECK ("duration_ms" >= 0 AND "finished_at" >= "started_at");

COMMENT ON TABLE "agent_analysis_runs" IS 'Every script an agent ran that showed the person a figure, table or chart: what it was for, the code, the datasets it read and where they came from, what it printed and raised, and what it showed';

COMMENT ON COLUMN "agent_analysis_runs"."datasets" IS 'The report runs and attached files the script read, each with the name it read it by and its row count; their rows are not kept here';

COMMENT ON COLUMN "agent_analysis_runs"."output" IS 'What the script printed, cut to the first 20,000 characters';

COMMENT ON COLUMN "agent_analysis_runs"."tables" IS 'Tables the script showed with show_table, kept as values so they are drawn as tables; computed from the datasets and reads the script made';

COMMENT ON COLUMN "agent_analysis_runs"."figures" IS 'Headline figures the script showed with show_figure, each with its label and kind';

COMMENT ON COLUMN "agent_analysis_runs"."charts" IS 'Charts the script showed with show_chart, kept as specs (kind, axis, series) the client draws';

COMMENT ON COLUMN "agent_analysis_runs"."layout" IS 'The order the script showed its charts and tables in: each a kind and its index among the charts or the tables';

DELETE FROM "agent_extension_usage_daily" WHERE "extension_type" = 'E2B';

DELETE FROM "agent_extensions" WHERE "type" = 'E2B';

ALTER TABLE "agent_extensions"
    DROP CONSTRAINT IF EXISTS "ck_agent_extensions_type";

ALTER TABLE "agent_extensions"
    ADD CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa', 'Parallel', 'BrowserUse'));

DELETE FROM "resource_permissions" WHERE "resource" = 'data_analysis';

DELETE FROM "api_key_permissions" WHERE "resource" = 'data_analysis';

UPDATE "agent_definitions"
SET "disabled_tool_names" = array_remove("disabled_tool_names", 'report_analyze_data')
WHERE 'report_analyze_data' = ANY("disabled_tool_names");

SELECT "agent_rename_tool"('report_analyze_data', 'agent_run_script');
