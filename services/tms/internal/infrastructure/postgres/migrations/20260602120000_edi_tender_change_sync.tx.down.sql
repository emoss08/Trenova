DELETE FROM "edi_source_context_fields"
WHERE "id" = 'ediscf_x12_204_purpose_code';

DROP INDEX IF EXISTS "idx_edi_tender_changes_search";
DROP INDEX IF EXISTS "idx_edi_tender_changes_source";
DROP INDEX IF EXISTS "idx_edi_tender_changes_actionable";
DROP INDEX IF EXISTS "idx_edi_tender_changes_idempotency";
DROP TABLE IF EXISTS "edi_tender_changes";

DROP INDEX IF EXISTS "idx_edi_tender_recipients_recipient_org";
DROP INDEX IF EXISTS "idx_edi_tender_recipients_source";
DROP INDEX IF EXISTS "idx_edi_tender_recipients_unique_recipient";
DROP TABLE IF EXISTS "edi_tender_recipients";

DROP INDEX IF EXISTS "idx_edi_messages_delivery";
ALTER TABLE "edi_messages"
    DROP COLUMN IF EXISTS "delivery_last_error",
    DROP COLUMN IF EXISTS "delivery_sent_at",
    DROP COLUMN IF EXISTS "delivery_last_attempt_at",
    DROP COLUMN IF EXISTS "delivery_attempts",
    DROP COLUMN IF EXISTS "delivery_remote_path",
    DROP COLUMN IF EXISTS "delivery_status";

DROP TYPE IF EXISTS edi_message_delivery_status_enum;
DROP TYPE IF EXISTS edi_tender_change_status_enum;
DROP TYPE IF EXISTS edi_tender_recipient_baseline_status_enum;
DROP TYPE IF EXISTS edi_tender_recipient_status_enum;
DROP TYPE IF EXISTS edi_tender_recipient_kind_enum;
