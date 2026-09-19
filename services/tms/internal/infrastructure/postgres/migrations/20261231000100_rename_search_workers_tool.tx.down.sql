-- Restores the old spelling so a rollback leaves agents able to resolve their
-- tools against the previous registry, which knows only "search_workers".

UPDATE "agent_definitions"
SET "tool_names" = array_replace("tool_names", 'search_worker', 'search_workers')
WHERE 'search_worker' = ANY("tool_names");

--bun:split
UPDATE "agent_definitions"
SET "tool_tiers" = ("tool_tiers" - 'search_worker')
    || jsonb_build_object('search_workers', "tool_tiers" -> 'search_worker')
WHERE "tool_tiers" ? 'search_worker'
  AND NOT ("tool_tiers" ? 'search_workers');

--bun:split
UPDATE "agent_definitions"
SET "tool_tiers" = "tool_tiers" - 'search_worker'
WHERE "tool_tiers" ? 'search_worker';
