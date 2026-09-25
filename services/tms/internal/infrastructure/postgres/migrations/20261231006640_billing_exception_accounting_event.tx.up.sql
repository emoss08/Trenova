-- The billing exception agent now also works an accounting connection that
-- needs attention. Only an agent still on the previous default moves; one whose
-- events somebody changed is left as it is.
UPDATE "agent_definitions"
SET "event_kinds" = ARRAY['billing_queue.item_exception', 'billing_queue.item_on_hold', 'accounting.connection_degraded']::TEXT[],
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
WHERE "system_key" = 'billing_exception'
  AND "event_kinds" = ARRAY['billing_queue.item_exception', 'billing_queue.item_on_hold']::TEXT[];
