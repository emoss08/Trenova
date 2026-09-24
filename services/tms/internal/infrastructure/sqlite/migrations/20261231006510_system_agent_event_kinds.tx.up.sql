-- Hand-written: SQLite keeps event_kinds as text, written as the Postgres
-- array literal (braces, with or without quoted elements) or as a JSON array
-- depending on how the row was saved, and has no array operators. A row is
-- moved only when its text is the seeded default in one of those spellings,
-- and keeps that spelling.
-- Source: 20261231006510_system_agent_event_kinds.tx.up.sql

UPDATE "agent_definitions"
SET "event_kinds" = CASE "event_kinds"
        WHEN '{billing_queue.item_exception}' THEN '{billing_queue.item_exception,billing_queue.item_on_hold}'
        WHEN '{"billing_queue.item_exception"}' THEN '{"billing_queue.item_exception","billing_queue.item_on_hold"}'
        WHEN '["billing_queue.item_exception"]' THEN '["billing_queue.item_exception","billing_queue.item_on_hold"]'
    END,
    "updated_at" = unixepoch()
WHERE "system_key" = 'billing_exception'
  AND "event_kinds" IN ('{billing_queue.item_exception}', '{"billing_queue.item_exception"}', '["billing_queue.item_exception"]');

--bun:split

UPDATE "agent_definitions"
SET "event_kinds" = CASE "event_kinds"
        WHEN '{shipment_move.coverage_at_risk}' THEN '{shipment_move.coverage_at_risk,shipment_move.unassigned}'
        WHEN '{"shipment_move.coverage_at_risk"}' THEN '{"shipment_move.coverage_at_risk","shipment_move.unassigned"}'
        WHEN '["shipment_move.coverage_at_risk"]' THEN '["shipment_move.coverage_at_risk","shipment_move.unassigned"]'
    END,
    "updated_at" = unixepoch()
WHERE "system_key" = 'dispatch_assignment'
  AND "event_kinds" IN ('{shipment_move.coverage_at_risk}', '{"shipment_move.coverage_at_risk"}', '["shipment_move.coverage_at_risk"]');
