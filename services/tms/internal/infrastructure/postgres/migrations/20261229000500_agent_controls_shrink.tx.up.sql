-- The per-agent switches that used to live on agent_controls now live on the
-- system agent definitions themselves. Carry each organization's choices over
-- before the columns go so nothing an operator set is lost.
UPDATE "agent_definitions" AS "agdef"
SET
    "enabled" = "agc"."billing_agent_enabled",
    "decision_timeout_seconds" = "agc"."decision_timeout_seconds",
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
FROM "agent_controls" AS "agc"
WHERE "agdef"."organization_id" = "agc"."organization_id"
  AND "agdef"."business_unit_id" = "agc"."business_unit_id"
  AND "agdef"."system_key" = 'billing_exception';

--bun:split
UPDATE "agent_definitions" AS "agdef"
SET
    "enabled" = "agc"."dispatch_agent_enabled",
    "autonomy_ceiling" = "agc"."dispatch_autonomy_tier",
    "tool_tiers" = COALESCE("agdef"."tool_tiers", '{}'::jsonb)
        || jsonb_build_object('assign_move', "agc"."dispatch_autonomy_tier"::text),
    "updated_at" = extract(epoch FROM current_timestamp)::bigint
FROM "agent_controls" AS "agc"
WHERE "agdef"."organization_id" = "agc"."organization_id"
  AND "agdef"."business_unit_id" = "agc"."business_unit_id"
  AND "agdef"."system_key" = 'dispatch_assignment';

--bun:split
ALTER TABLE "agent_controls"
    DROP CONSTRAINT IF EXISTS "chk_agent_controls_decision_timeout";

--bun:split
ALTER TABLE "agent_controls"
    DROP COLUMN IF EXISTS "billing_agent_enabled",
    DROP COLUMN IF EXISTS "dispatch_agent_enabled",
    DROP COLUMN IF EXISTS "dispatch_autonomy_tier",
    DROP COLUMN IF EXISTS "decision_timeout_seconds";
