ALTER TABLE "agent_controls"
    ADD COLUMN IF NOT EXISTS "billing_agent_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "dispatch_agent_enabled" BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS "dispatch_autonomy_tier" agent_autonomy_tier_enum NOT NULL DEFAULT 'Propose',
    ADD COLUMN IF NOT EXISTS "decision_timeout_seconds" INTEGER NOT NULL DEFAULT 86400;

--bun:split
ALTER TABLE "agent_controls"
    ADD CONSTRAINT "chk_agent_controls_decision_timeout" CHECK ("decision_timeout_seconds" >= 60);

--bun:split
UPDATE "agent_controls" AS "agc"
SET
    "billing_agent_enabled" = "agdef"."enabled",
    "decision_timeout_seconds" = "agdef"."decision_timeout_seconds"
FROM "agent_definitions" AS "agdef"
WHERE "agdef"."organization_id" = "agc"."organization_id"
  AND "agdef"."business_unit_id" = "agc"."business_unit_id"
  AND "agdef"."system_key" = 'billing_exception';

--bun:split
UPDATE "agent_controls" AS "agc"
SET
    "dispatch_agent_enabled" = "agdef"."enabled",
    "dispatch_autonomy_tier" = "agdef"."autonomy_ceiling"
FROM "agent_definitions" AS "agdef"
WHERE "agdef"."organization_id" = "agc"."organization_id"
  AND "agdef"."business_unit_id" = "agc"."business_unit_id"
  AND "agdef"."system_key" = 'dispatch_assignment';
