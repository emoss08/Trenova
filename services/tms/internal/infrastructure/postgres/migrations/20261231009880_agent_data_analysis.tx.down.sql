DELETE FROM "assistant_artifacts" WHERE "kind" = 'analysis';

--bun:split
ALTER TABLE "assistant_artifacts"
    DROP CONSTRAINT IF EXISTS "ck_assistant_artifacts_kind";

--bun:split
ALTER TABLE "assistant_artifacts"
    ADD CONSTRAINT "ck_assistant_artifacts_kind" CHECK (
        "kind" IN (
            'report_preview', 'report_run', 'email_draft', 'plan', 'entity_card',
            'table_view', 'rate_explanation', 'dashboard_ref', 'briefing',
            'inbound_message', 'run_diff', 'document', 'navigation', 'draft_edit',
            'decision_request', 'extraction', 'edi_connection_draft'
        )
    );

--bun:split
DROP TABLE IF EXISTS "agent_analysis_charts";

--bun:split
DROP TABLE IF EXISTS "agent_analysis_runs";

--bun:split
DELETE FROM "agent_extension_usage_daily" WHERE "extension_type" = 'E2B';

--bun:split
DELETE FROM "agent_extensions" WHERE "type" = 'E2B';

--bun:split
ALTER TABLE "agent_extensions"
    DROP CONSTRAINT IF EXISTS "ck_agent_extensions_type";

--bun:split
ALTER TABLE "agent_extensions"
    ADD CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa', 'Parallel', 'BrowserUse'));
