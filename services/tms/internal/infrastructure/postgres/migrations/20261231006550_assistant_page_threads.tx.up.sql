-- An import or a formula conversation lives on the page of its subject: the
-- document being turned into a shipment, or the formula template being
-- written. There is one live conversation per person and subject, and the
-- Desk never lists it.
ALTER TABLE "assistant_threads"
    DROP CONSTRAINT IF EXISTS "ck_assistant_threads_origin";

--bun:split
ALTER TABLE "assistant_threads"
    ADD CONSTRAINT "ck_assistant_threads_origin" CHECK (
        "origin" IN ('Panel', 'Desk', 'Ask', 'Watchtower', 'Briefing', 'Import', 'Formula')
    );

--bun:split
-- Two tabs opening the same document at once must land in the same
-- conversation, so the page's conversation is claimed on this key rather
-- than found by a read that both could pass.
CREATE UNIQUE INDEX IF NOT EXISTS "uq_assistant_threads_page_subject" ON "assistant_threads"(
    "organization_id", "business_unit_id", "origin", "subject_type", "subject_id", "user_id"
)
WHERE "origin" IN ('Import', 'Formula')
    AND "status" = 'Active'
    AND "subject_id" IS NOT NULL;

--bun:split
-- A change an assistant hands a page to apply to its unsaved work, kept so
-- the conversation still says what it changed after a reload.
ALTER TABLE "assistant_artifacts"
    DROP CONSTRAINT IF EXISTS "ck_assistant_artifacts_kind";

--bun:split
ALTER TABLE "assistant_artifacts"
    ADD CONSTRAINT "ck_assistant_artifacts_kind" CHECK (
        "kind" IN (
            'report_preview', 'report_run', 'email_draft', 'plan', 'entity_card',
            'table_view', 'rate_explanation', 'dashboard_ref', 'briefing',
            'inbound_message', 'run_diff', 'document', 'navigation', 'draft_edit'
        )
    );

COMMENT ON INDEX "uq_assistant_threads_page_subject" IS 'One live import or formula conversation per person and subject';
