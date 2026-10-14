--
-- Copyright 2023-2025 Eric Moss
-- Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
-- Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md--

DROP TABLE IF EXISTS "ai_analytics_refresh_state";

--bun:split
DROP TABLE IF EXISTS "ai_provider_hourly_stats";

--bun:split
DROP TABLE IF EXISTS "ai_work_unit_tools";

--bun:split
DROP TABLE IF EXISTS "ai_work_units";

--bun:split
ALTER TABLE "agent_run_steps" DROP COLUMN IF EXISTS "settled_at_ms";

--bun:split
ALTER TABLE "agent_run_steps" DROP COLUMN IF EXISTS "started_at_ms";

--bun:split
ALTER TABLE "ai_usage_records" DROP COLUMN IF EXISTS "recorded_at_ms";
