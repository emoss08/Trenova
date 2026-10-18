ALTER TABLE "agent_analysis_runs"
    ADD COLUMN IF NOT EXISTS "title" VARCHAR(80) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS "tables" JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "figures" JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS "layout" JSONB NOT NULL DEFAULT '[]'::jsonb;

--bun:split
COMMENT ON COLUMN "agent_analysis_runs"."tables" IS 'Tables the code showed with show_table, kept as values so they are drawn as tables; computed from the datasets the analysis read';

--bun:split
COMMENT ON COLUMN "agent_analysis_runs"."layout" IS 'The order the code showed its charts and tables in: each a chart position or a table index';
