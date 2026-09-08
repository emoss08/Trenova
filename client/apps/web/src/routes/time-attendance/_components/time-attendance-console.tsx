import { usePermission } from "@/hooks/use-permission";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { startOfRotaWeek } from "@trenova/shared/lib/scheduling";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { BanknoteIcon, ClipboardCheckIcon, TimerIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { PayrollPanel } from "./payroll-panel";
import {
  awaitingApprovalQuery,
  openEntriesQuery,
  thisWeekQuery,
  unpaidApprovedQuery,
} from "./queries";
import { TimeAttendanceOverview } from "./time-attendance-overview";
import { TimeClockPanel } from "./time-clock-panel";
import { TimesheetQueue } from "./timesheet-queue";

const CLOCK_TICK_MS = 30_000;

export default function TimeAttendanceConsole() {
  const { allowed: canRead } = usePermission(Resource.Timesheet, Operation.Read);
  const { allowed: canExport } = usePermission(Resource.Timesheet, Operation.Export);
  const [teamOnly, setTeamOnly] = useState(false);
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  // Running totals are what somebody watches while they are on the clock, so
  // the page ticks rather than waiting for the next request.
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Math.floor(Date.now() / 1000)), CLOCK_TICK_MS);
    return () => window.clearInterval(timer);
  }, []);

  const weekStart = startOfRotaWeek(now);
  const awaiting = useQuery({ ...awaitingApprovalQuery(), enabled: canRead });
  const thisWeek = useQuery({ ...thisWeekQuery(weekStart), enabled: canRead });
  const unpaid = useQuery({ ...unpaidApprovedQuery(), enabled: canRead && canExport });
  const openEntries = useQuery({ ...openEntriesQuery(teamOnly), enabled: canRead });

  if (!canRead) return null;

  const awaitingCount = awaiting.data?.length ?? 0;
  const onClockCount = openEntries.data?.length ?? 0;

  return (
    <div className="flex flex-col gap-4">
      <TimeAttendanceOverview
        awaiting={awaiting.data}
        thisWeek={thisWeek.data}
        unpaid={unpaid.data}
        openEntries={openEntries.data}
        now={now}
        showPayroll={canExport}
      />

      <Tabs defaultValue="clock">
        <TabsList variant="underline">
          <TabsTrigger value="clock">
            <TimerIcon className="size-3.5" />
            Clock
            {onClockCount > 0 ? (
              <Badge variant="secondary" className="text-2xs ml-1.5 h-4 px-1 tabular-nums">
                {onClockCount}
              </Badge>
            ) : null}
          </TabsTrigger>
          <TabsTrigger value="timesheets">
            <ClipboardCheckIcon className="size-3.5" />
            Timesheets
            {awaitingCount > 0 ? (
              <Badge variant="warning" className="text-2xs ml-1.5 h-4 px-1 tabular-nums">
                {awaitingCount}
              </Badge>
            ) : null}
          </TabsTrigger>
          {canExport ? (
            <TabsTrigger value="payroll">
              <BanknoteIcon className="size-3.5" />
              Payroll
            </TabsTrigger>
          ) : null}
        </TabsList>

        <TabsContent value="clock">
          <TimeClockPanel
            now={now}
            openEntries={openEntries.data}
            openEntriesLoading={openEntries.isLoading}
            teamOnly={teamOnly}
            onTeamOnlyChange={setTeamOnly}
          />
        </TabsContent>
        <TabsContent value="timesheets">
          <TimesheetQueue now={now} />
        </TabsContent>
        {canExport ? (
          <TabsContent value="payroll">
            <PayrollPanel />
          </TabsContent>
        ) : null}
      </Tabs>
    </div>
  );
}
