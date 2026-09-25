-- Hand-written: reverses 20261231006660_billing_exception_accounting_event.tx.up.sql
-- in each spelling SQLite may hold event_kinds in.
-- Source: 20261231006660_billing_exception_accounting_event.tx.down.sql

UPDATE "agent_definitions"
SET "event_kinds" = CASE "event_kinds"
        WHEN '{billing_queue.item_exception,billing_queue.item_on_hold,accounting.connection_degraded}' THEN '{billing_queue.item_exception,billing_queue.item_on_hold}'
        WHEN '{"billing_queue.item_exception","billing_queue.item_on_hold","accounting.connection_degraded"}' THEN '{"billing_queue.item_exception","billing_queue.item_on_hold"}'
        WHEN '["billing_queue.item_exception","billing_queue.item_on_hold","accounting.connection_degraded"]' THEN '["billing_queue.item_exception","billing_queue.item_on_hold"]'
    END,
    "updated_at" = unixepoch()
WHERE "system_key" = 'billing_exception'
  AND "event_kinds" IN ('{billing_queue.item_exception,billing_queue.item_on_hold,accounting.connection_degraded}', '{"billing_queue.item_exception","billing_queue.item_on_hold","accounting.connection_degraded"}', '["billing_queue.item_exception","billing_queue.item_on_hold","accounting.connection_degraded"]');
