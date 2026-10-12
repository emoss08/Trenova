DROP TABLE IF EXISTS "inbound_sender_hints";

--bun:split
DROP TABLE IF EXISTS "triage_views";

--bun:split
DROP TABLE IF EXISTS "triage_settings";

--bun:split
DROP TABLE IF EXISTS "triage_response_targets";

--bun:split
DROP TABLE IF EXISTS "triage_reply_templates";

--bun:split
DROP TABLE IF EXISTS "triage_rules";

--bun:split
DROP TABLE IF EXISTS "inbound_replies";

--bun:split
DROP TABLE IF EXISTS "inbound_message_activities";

--bun:split
ALTER TABLE "inbound_mailboxes"
    DROP COLUMN IF EXISTS "blocked_senders",
    DROP COLUMN IF EXISTS "test_received_at",
    DROP COLUMN IF EXISTS "test_sent_at",
    DROP COLUMN IF EXISTS "test_token",
    DROP COLUMN IF EXISTS "last_received_at";

--bun:split
ALTER TABLE "inbound_message_attachments"
    DROP COLUMN IF EXISTS "split_documents",
    DROP COLUMN IF EXISTS "pages",
    DROP COLUMN IF EXISTS "page_count",
    DROP COLUMN IF EXISTS "filed_by",
    DROP COLUMN IF EXISTS "filed_at",
    DROP COLUMN IF EXISTS "filed_as",
    DROP COLUMN IF EXISTS "filed_shipment_id";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_settled_at";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_rule";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_sender";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_kind_due";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_mailbox_lane";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_message_key";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_thread_key";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_owner";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_snooze_event";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_awaiting_until";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_snoozed_until";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_parked";

--bun:split
DROP INDEX IF EXISTS "idx_inbound_messages_waiting_due";

--bun:split
ALTER TABLE "inbound_messages"
    DROP CONSTRAINT IF EXISTS "ck_inbound_messages_parked_needs_review",
    DROP CONSTRAINT IF EXISTS "ck_inbound_messages_handled_action",
    DROP CONSTRAINT IF EXISTS "ck_inbound_messages_snooze_event",
    DROP CONSTRAINT IF EXISTS "ck_inbound_messages_parked";

--bun:split
ALTER TABLE "inbound_messages"
    DROP COLUMN IF EXISTS "load_draft",
    DROP COLUMN IF EXISTS "action_state",
    DROP COLUMN IF EXISTS "conflicts",
    DROP COLUMN IF EXISTS "facts",
    DROP COLUMN IF EXISTS "classification_corrected",
    DROP COLUMN IF EXISTS "handled_action",
    DROP COLUMN IF EXISTS "handled_by_rule_id",
    DROP COLUMN IF EXISTS "nova_read",
    DROP COLUMN IF EXISTS "awaiting_until",
    DROP COLUMN IF EXISTS "awaiting_since",
    DROP COLUMN IF EXISTS "snoozed_by",
    DROP COLUMN IF EXISTS "snoozed_at",
    DROP COLUMN IF EXISTS "snooze_document_type_id",
    DROP COLUMN IF EXISTS "snooze_watch_id",
    DROP COLUMN IF EXISTS "snooze_event",
    DROP COLUMN IF EXISTS "snoozed_until",
    DROP COLUMN IF EXISTS "parked",
    DROP COLUMN IF EXISTS "due_at",
    DROP COLUMN IF EXISTS "owner_id",
    DROP COLUMN IF EXISTS "sender_domain",
    DROP COLUMN IF EXISTS "thread_key",
    DROP COLUMN IF EXISTS "message_key";
