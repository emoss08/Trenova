DELETE FROM "assistant_artifacts"
WHERE "kind" = 'draft_edit';

--bun:split
ALTER TABLE "assistant_artifacts"
    DROP CONSTRAINT IF EXISTS "ck_assistant_artifacts_kind";

--bun:split
ALTER TABLE "assistant_artifacts"
    ADD CONSTRAINT "ck_assistant_artifacts_kind" CHECK (
        "kind" IN (
            'report_preview', 'report_run', 'email_draft', 'plan', 'entity_card',
            'table_view', 'rate_explanation', 'dashboard_ref', 'briefing',
            'inbound_message', 'run_diff', 'document', 'navigation'
        )
    );

--bun:split
DROP INDEX IF EXISTS "uq_assistant_threads_page_subject";

--bun:split
DELETE FROM "assistant_threads"
WHERE "origin" IN ('Import', 'Formula');

--bun:split
ALTER TABLE "assistant_threads"
    DROP CONSTRAINT IF EXISTS "ck_assistant_threads_origin";

--bun:split
ALTER TABLE "assistant_threads"
    ADD CONSTRAINT "ck_assistant_threads_origin" CHECK (
        "origin" IN ('Panel', 'Desk', 'Ask', 'Watchtower', 'Briefing')
    );
