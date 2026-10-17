DELETE FROM "agent_waits" WHERE "kind" = 'DocumentRead';

--bun:split
ALTER TABLE "agent_waits"
    DROP CONSTRAINT IF EXISTS "ck_agent_waits_kind";

--bun:split
ALTER TABLE "agent_waits"
    ADD CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow', 'PortalRun'));
