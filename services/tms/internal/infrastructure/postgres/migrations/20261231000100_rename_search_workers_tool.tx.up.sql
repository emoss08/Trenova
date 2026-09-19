-- The agent tool "search_workers" is now "search_worker".
--
-- A tool name is not only a symbol: it is stored in agent_definitions.tool_names
-- and used as a key in tool_tiers, so every agent an organization has already
-- saved names the tool by its old spelling. Without this rewrite those agents
-- would resolve nothing for that entry, and the failure would be silent — the
-- agent simply stops being able to look drivers up.
--
-- The registry also keeps a permanent alias for the old name, which covers a
-- definition written by an API client that hard-coded it after this ran.

UPDATE "agent_definitions"
SET "tool_names" = array_replace("tool_names", 'search_workers', 'search_worker')
WHERE 'search_workers' = ANY("tool_names");

--bun:split
-- A per-tool autonomy override is keyed by tool name. Moving the value and
-- dropping the old key keeps the override attached to the tool it was set for;
-- the guard below leaves an object that somehow holds both keys alone rather
-- than picking a winner.
UPDATE "agent_definitions"
SET "tool_tiers" = ("tool_tiers" - 'search_workers')
    || jsonb_build_object('search_worker', "tool_tiers" -> 'search_workers')
WHERE "tool_tiers" ? 'search_workers'
  AND NOT ("tool_tiers" ? 'search_worker');

--bun:split
-- An object carrying both keys predates nothing we wrote, but dropping the
-- stale one is still right: the tool it named no longer exists.
UPDATE "agent_definitions"
SET "tool_tiers" = "tool_tiers" - 'search_workers'
WHERE "tool_tiers" ? 'search_workers';
