-- An agent that takes somebody to a page leaves a card saying where it took
-- them, so the conversation still shows it after a reload.
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
