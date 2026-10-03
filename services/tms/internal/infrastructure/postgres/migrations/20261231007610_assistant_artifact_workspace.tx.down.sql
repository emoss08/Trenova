DROP INDEX IF EXISTS "idx_assistant_artifacts_lineage_root";

--bun:split
DROP INDEX IF EXISTS "uq_assistant_artifacts_slug";

--bun:split
ALTER TABLE "assistant_artifacts"
    DROP COLUMN IF EXISTS "slug";

--bun:split
DELETE FROM "assistant_artifacts" WHERE "kind" = 'extraction';

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
            'decision_request'
        )
    );
