ALTER TABLE "agent_definitions"
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_business_hours",
    DROP CONSTRAINT IF EXISTS "ck_agent_definitions_max_change_items";

ALTER TABLE "agent_definitions"
    DROP COLUMN IF EXISTS "created_by_id",
    DROP COLUMN IF EXISTS "max_change_items",
    DROP COLUMN IF EXISTS "business_hours_only",
    DROP COLUMN IF EXISTS "business_hours_start",
    DROP COLUMN IF EXISTS "business_hours_end",
    DROP COLUMN IF EXISTS "business_hours_timezone",
    DROP COLUMN IF EXISTS "delegate_topics",
    DROP COLUMN IF EXISTS "disabled_tool_names";
