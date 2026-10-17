DELETE FROM "agent_waits" WHERE "kind" = 'PortalRun';

--bun:split
ALTER TABLE "agent_waits"
    DROP CONSTRAINT IF EXISTS "ck_agent_waits_kind";

--bun:split
ALTER TABLE "agent_waits"
    ADD CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow'));

--bun:split
DROP TABLE IF EXISTS "agent_portal_runs";

--bun:split
DROP TABLE IF EXISTS "agent_portal_sites";

--bun:split
DELETE FROM "agent_extension_usage_daily" WHERE "extension_type" = 'BrowserUse';

--bun:split
DELETE FROM "agent_extensions" WHERE "type" = 'BrowserUse';

--bun:split
ALTER TABLE "agent_extensions"
    DROP CONSTRAINT IF EXISTS "ck_agent_extensions_type";

--bun:split
ALTER TABLE "agent_extensions"
    ADD CONSTRAINT "ck_agent_extensions_type" CHECK ("type" IN ('Exa', 'Parallel'));
