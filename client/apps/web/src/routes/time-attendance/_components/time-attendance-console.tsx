import { usePermission } from "@/hooks/use-permission";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { PayrollPanel } from "./payroll-panel";
import { TimeClockPanel } from "./time-clock-panel";
import { TimesheetQueue } from "./timesheet-queue";

export default function TimeAttendanceConsole() {
  const { allowed: canRead } = usePermission(Resource.Timesheet, Operation.Read);
  const { allowed: canExport } = usePermission(Resource.Timesheet, Operation.Export);

  if (!canRead) return null;

  return (
    <Tabs defaultValue="clock">
      <TabsList>
        <TabsTrigger value="clock">Clock</TabsTrigger>
        <TabsTrigger value="timesheets">Timesheets</TabsTrigger>
        {canExport ? <TabsTrigger value="payroll">Payroll</TabsTrigger> : null}
      </TabsList>

      <TabsContent value="clock">
        <TimeClockPanel />
      </TabsContent>
      <TabsContent value="timesheets">
        <TimesheetQueue />
      </TabsContent>
      {canExport ? (
        <TabsContent value="payroll">
          <PayrollPanel />
        </TabsContent>
      ) : null}
    </Tabs>
  );
}
