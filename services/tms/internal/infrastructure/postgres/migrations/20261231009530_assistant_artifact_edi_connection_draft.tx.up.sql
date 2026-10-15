-- An EDI connection drafted from a trading partner's guide is reviewed as an
-- artifact of its own: every value with its page, what was held for a person,
-- and the records the approval would write.
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
