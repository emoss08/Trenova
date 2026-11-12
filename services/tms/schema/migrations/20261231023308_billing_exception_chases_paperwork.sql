-- A delivered load with no proof of delivery cannot be billed, and the
-- billing exception agent is the one that chases it when its organization
-- has it listen for "Proof of delivery missing". Chasing means asking the
-- driver who delivered it or emailing the carrier, so the agent is given the
-- tools for both. Its events are left as they are: listening for the new one
-- is switched on per agent in AI Control.
UPDATE "agent_definitions"
SET "tool_names" = "tool_names" || ARRAY(
        SELECT tool
        FROM unnest(ARRAY['inbox_list_email_profiles', 'worker_get', 'worker_send_message']::varchar[]) AS tool
        WHERE NOT (tool = ANY ("tool_names")))
WHERE "system_key" = 'billing_exception'
    AND cardinality("tool_names") > 0;
