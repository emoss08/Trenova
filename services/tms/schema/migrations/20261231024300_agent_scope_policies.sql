-- An agent is a policy, not a tool list: a scope of areas and capabilities allowed or denied
-- (empty is everything, which is the universal agent), at most twelve pinned capabilities
-- that ride on every turn, and approval overrides in tool_tiers. Existing agents are not
-- converted: each is switched off with a scope that denies everything but the core tools,
-- and `trenova-cli agents reset` archives them and creates the default desks again.
ALTER TABLE "agent_definitions"
    ADD COLUMN IF NOT EXISTS "scope" jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS "pinned_tools" text[];

UPDATE
    "agent_definitions"
SET
    "scope" = '{"denyAreas": ["*"]}'::jsonb,
    "enabled" = FALSE,
    "disabled_at" = COALESCE("disabled_at", extract(epoch FROM now())::bigint),
    "tool_tiers" = NULL,
    "tool_daily_limits" = '{}'::jsonb;

ALTER TABLE "agent_definitions"
    DROP COLUMN IF EXISTS "tool_names",
    DROP COLUMN IF EXISTS "disabled_tool_names",
    ADD CONSTRAINT "ck_agent_definitions_scope_object" CHECK (jsonb_typeof("scope") = 'object'),
    ADD CONSTRAINT "ck_agent_definitions_pinned_tools" CHECK (coalesce(cardinality("pinned_tools"), 0) <= 12);

-- Tool names are no longer renamed in saved data: a removed capability is cleared, not mapped.
DROP FUNCTION IF EXISTS "agent_rename_tool"(text, text);
