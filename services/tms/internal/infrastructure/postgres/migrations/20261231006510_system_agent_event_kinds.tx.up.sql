UPDATE "agent_definitions"
SET "event_kinds" = ARRAY['billing_queue.item_exception', 'billing_queue.item_on_hold']::TEXT[],
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
WHERE "system_key" = 'billing_exception'
  AND "event_kinds" = ARRAY['billing_queue.item_exception']::TEXT[];

--bun:split

UPDATE "agent_definitions"
SET "event_kinds" = ARRAY['shipment_move.coverage_at_risk', 'shipment_move.unassigned']::TEXT[],
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
WHERE "system_key" = 'dispatch_assignment'
  AND "event_kinds" = ARRAY['shipment_move.coverage_at_risk']::TEXT[];
