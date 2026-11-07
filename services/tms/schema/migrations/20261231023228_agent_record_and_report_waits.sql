-- A conversation can park on a record changing (RecordChanged: a person fixing what
-- blocked a write) and on a report run it started finishing (ReportRun).
ALTER TABLE "agent_waits"
    DROP CONSTRAINT IF EXISTS "ck_agent_waits_kind";

ALTER TABLE "agent_waits"
    ADD CONSTRAINT "ck_agent_waits_kind" CHECK ("kind" IN ('Time', 'StopArrival', 'StopDeparture', 'Reply', 'AppointmentNear', 'FreeTimeEnding', 'HOSDriveBelow', 'PortalRun', 'DocumentRead', 'WebWatch', 'RecordChanged', 'ReportRun'));
