DELETE FROM "agent_waits" WHERE "kind" = 'WebWatch';

--bun:split
ALTER TABLE "agent_waits"
    DROP CONSTRAINT IF EXISTS "ck_agent_waits_kind";

--bun:split
ALTER TABLE "agent_waits"
    ADD CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow', 'PortalRun', 'DocumentRead'));

--bun:split
DROP TABLE IF EXISTS "agent_web_watch_findings";

--bun:split
DROP TABLE IF EXISTS "agent_web_watches";
