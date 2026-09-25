-- Hand-written: SQLite keeps event_kinds as text, written as the Postgres
-- array literal (braces, with or without quoted elements) or as a JSON array
-- depending on how the row was saved, and has no array operators. A row is
-- moved only when its text is the previous default in one of those spellings,
-- and keeps that spelling.
-- Source: 20261231006640_billing_exception_accounting_event.tx.up.sql

UPDATE "agent_definitions"
SET "event_kinds" = CASE "event_kinds"
        WHEN '{billing_queue.item_exception,billing_queue.item_on_hold}' THEN '{billing_queue.item_exception,billing_queue.item_on_hold,accounting.connection_degraded}'
        WHEN '{"billing_queue.item_exception","billing_queue.item_on_hold"}' THEN '{"billing_queue.item_exception","billing_queue.item_on_hold","accounting.connection_degraded"}'
        WHEN '["billing_queue.item_exception","billing_queue.item_on_hold"]' THEN '["billing_queue.item_exception","billing_queue.item_on_hold","accounting.connection_degraded"]'
    END,
    "updated_at" = unixepoch()
WHERE "system_key" = 'billing_exception'
  AND "event_kinds" IN ('{billing_queue.item_exception,billing_queue.item_on_hold}', '{"billing_queue.item_exception","billing_queue.item_on_hold"}', '["billing_queue.item_exception","billing_queue.item_on_hold"]');
