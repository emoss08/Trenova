-- SQLite twin of 20261231008800_triage.tx.down.sql.

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

ALTER TABLE "inbound_mailboxes" DROP COLUMN "blocked_senders";

--bun:split

ALTER TABLE "inbound_mailboxes" DROP COLUMN "test_received_at";

--bun:split

ALTER TABLE "inbound_mailboxes" DROP COLUMN "test_sent_at";

--bun:split

ALTER TABLE "inbound_mailboxes" DROP COLUMN "test_token";

--bun:split

ALTER TABLE "inbound_mailboxes" DROP COLUMN "last_received_at";

--bun:split

ALTER TABLE "inbound_message_attachments" DROP COLUMN "split_documents";

--bun:split

ALTER TABLE "inbound_message_attachments" DROP COLUMN "pages";

--bun:split

ALTER TABLE "inbound_message_attachments" DROP COLUMN "page_count";

--bun:split

ALTER TABLE "inbound_message_attachments" DROP COLUMN "filed_by";

--bun:split

ALTER TABLE "inbound_message_attachments" DROP COLUMN "filed_at";

--bun:split

ALTER TABLE "inbound_message_attachments" DROP COLUMN "filed_as";

--bun:split

ALTER TABLE "inbound_message_attachments" DROP COLUMN "filed_shipment_id";

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

ALTER TABLE "inbound_messages" DROP COLUMN "load_draft";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "action_state";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "conflicts";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "facts";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "classification_corrected";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "handled_action";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "handled_by_rule_id";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "nova_read";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "awaiting_until";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "awaiting_since";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "snoozed_by";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "snoozed_at";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "snooze_document_type_id";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "snooze_watch_id";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "snooze_event";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "snoozed_until";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "parked";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "due_at";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "owner_id";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "sender_domain";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "thread_key";

--bun:split

ALTER TABLE "inbound_messages" DROP COLUMN "message_key";
