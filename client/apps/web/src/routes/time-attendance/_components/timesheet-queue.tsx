import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { SegmentedBar } from "@/components/kpi/segmented-bar";
import { toneVar } from "@/components/kpi/tone";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchTimesheet,
  fetchTimesheets,
  TIMESHEETS_KEY,
  TIMESHEET_KEY,
  transitionTimesheet,
  type TimesheetRow,
} from "@/lib/graphql/timesheet";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@trenova/shared/components/ui/sheet";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { formatShiftDate } from "@trenova/shared/lib/scheduling";
import {
  formatHours,
  timesheetActionsFor,
  timesheetStatusTone,
} from "@trenova/shared/lib/timesheet";
import { initials } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  CheckIcon,
  ClipboardCheckIcon,
  ClockIcon,
  SendIcon,
  UndoIcon,
  UsersIcon,
} from "lucide-react";
import { useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";

type StatusFilter = "Submitted" | "Open" | "Approved" | "Locked";
type FilterValues = { workerId: string };

const STATUS_ITEMS = [
  { value: "Submitted", label: "Awaiting approval" },
  { value: "Open", label: "Open" },
  { value: "Approved", label: "Approved" },
  { value: "Locked", label: "Paid" },
] satisfies { value: StatusFilter; label: string }[];

const REGULAR = toneVar("brand");
const OVERTIME = toneVar("warning");
const LEAVE = toneVar("info");

function hourSegments(sheet: {
  regularMinutes: number;
  overtimeMinutes: number;
  paidLeaveMinutes: number;
}) {
  return [
    { label: "Regular", value: sheet.regularMinutes, color: REGULAR },
    { label: "Overtime", value: sheet.overtimeMinutes, color: OVERTIME },
    { label: "Paid leave", value: sheet.paidLeaveMinutes, color: LEAVE },
  ];
}

function formatPunchTime(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, { hour: "numeric", minute: "2-digit" });
}

/**
 * The approval queue. Submitted weeks come first because they are the ones
 * waiting on somebody, and a week's totals are what was frozen at submit —
 * approving it is approving those numbers, not whatever the punches say now.
 */
export function TimesheetQueue() {
  const queryClient = useQueryClient();
  const { allowed: canApprove } = usePermission(Resource.Timesheet, Operation.Approve);
  const { allowed: canSubmit } = usePermission(Resource.Timesheet, Operation.Submit);
  const [status, setStatus] = useState<StatusFilter>("Submitted");
  const [teamOnly, setTeamOnly] = useState(false);
  const [openSheetId, setOpenSheetId] = useState<string | null>(null);

  const filterForm = useForm<FilterValues>({ defaultValues: { workerId: "" } });
  const workerId = useWatch({ control: filterForm.control, name: "workerId" });

  const sheets = useQuery({
    queryKey: [TIMESHEETS_KEY, status, teamOnly, workerId],
    queryFn: ({ signal }) =>
      fetchTimesheets({ statuses: [status], teamOnly, workerId: workerId || null }, { signal }),
  });

  const { mutate: decide, isPending } = useMutation({
    mutationFn: (input: { id: string; status: "Submitted" | "Approved" | "Rejected" }) =>
      transitionTimesheet(input),
    onSuccess: (_data, input) => {
      toast.success(
        input.status === "Approved"
          ? "Week approved"
          : input.status === "Rejected"
            ? "Week sent back"
            : "Week handed over",
      );
      void queryClient.invalidateQueries({ queryKey: [TIMESHEETS_KEY] });
      void queryClient.invalidateQueries({ queryKey: [TIMESHEET_KEY] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const rows = sheets.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <SegmentedControl<StatusFilter>
          items={STATUS_ITEMS}
          value={status}
          onValueChange={setStatus}
          aria-label="Timesheet status"
        />
        <div className="flex flex-wrap items-end gap-2">
          <FormProvider {...filterForm}>
            <Form onSubmit={(event) => event.preventDefault()}>
              <FormGroup cols={1}>
                <FormControl className="w-56">
                  <WorkerAutocompleteField<FilterValues>
                    control={filterForm.control}
                    name="workerId"
                    placeholder="Any worker"
                    clearable
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </FormProvider>
          <Button
            size="sm"
            variant={teamOnly ? "default" : "outline"}
            onClick={() => setTeamOnly((value) => !value)}
            aria-pressed={teamOnly}
          >
            <UsersIcon className="size-3.5" />
            My team
          </Button>
        </div>
      </div>

      {sheets.isLoading ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-16 rounded-lg" />
          <Skeleton className="h-16 rounded-lg" />
          <Skeleton className="h-16 rounded-lg" />
        </div>
      ) : rows.length === 0 ? (
        <EmptyState
          className="max-w-none"
          title={`Nothing ${timesheetStatusTone(status).label.toLowerCase()}`}
          description={
            status === "Submitted"
              ? "Weeks handed over for approval land here."
              : "Try another status, or widen the filter."
          }
          icons={[ClipboardCheckIcon, ClockIcon, CheckIcon]}
        />
      ) : (
        <ul className="flex flex-col gap-2">
          {rows.map((sheet) => (
            <QueueRow
              key={sheet.id}
              sheet={sheet}
              busy={isPending}
              canApprove={canApprove}
              canSubmit={canSubmit}
              onOpen={() => setOpenSheetId(sheet.id)}
              onDecide={(next) => decide({ id: sheet.id, status: next })}
            />
          ))}
        </ul>
      )}

      <TimesheetSheet id={openSheetId} onOpenChange={(open) => !open && setOpenSheetId(null)} />
    </div>
  );
}

function QueueRow({
  sheet,
  busy,
  canApprove,
  canSubmit,
  onOpen,
  onDecide,
}: {
  sheet: TimesheetRow;
  busy: boolean;
  canApprove: boolean;
  canSubmit: boolean;
  onOpen: () => void;
  onDecide: (status: "Submitted" | "Approved" | "Rejected") => void;
}) {
  const tone = timesheetStatusTone(sheet.status);
  const actions = timesheetActionsFor(sheet.status, { isOwner: false, canApprove });
  const name = sheet.worker ? `${sheet.worker.firstName} ${sheet.worker.lastName}` : sheet.workerId;

  return (
    <li className="border-border/80 hover:border-border flex flex-wrap items-center gap-4 rounded-lg border p-3 text-xs transition-colors">
      <button
        type="button"
        className="flex min-w-0 flex-1 items-center gap-3 text-left"
        onClick={onOpen}
      >
        <Avatar className="size-8">
          <AvatarFallback className="text-[10px] font-medium">
            {sheet.worker ? initials(sheet.worker.firstName, sheet.worker.lastName) : "?"}
          </AvatarFallback>
        </Avatar>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{name}</span>
            <span className="text-muted-foreground tabular-nums">
              Week of {formatShiftDate(sheet.periodStart)}
            </span>
            <Badge variant={tone.variant}>{tone.label}</Badge>
          </span>
          <div className="max-w-md">
            <SegmentedBar segments={hourSegments(sheet)} />
          </div>
        </div>
      </button>

      <div className="flex shrink-0 items-center gap-3">
        <div className="text-right leading-tight">
          <p className="font-mono text-base font-semibold tabular-nums">
            {formatHours(sheet.totalMinutes)}
          </p>
          <p className="text-muted-foreground tabular-nums">
            {formatHours(sheet.regularMinutes)} reg
            {sheet.overtimeMinutes > 0 ? ` · ${formatHours(sheet.overtimeMinutes)} OT` : ""}
            {sheet.paidLeaveMinutes > 0 ? ` · ${formatHours(sheet.paidLeaveMinutes)} leave` : ""}
          </p>
        </div>
        {actions.includes("reject") ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onDecide("Rejected")}>
            <UndoIcon className="size-3.5" />
            Send back
          </Button>
        ) : null}
        {actions.includes("approve") ? (
          <Button size="sm" disabled={busy} onClick={() => onDecide("Approved")}>
            <CheckIcon className="size-3.5" />
            Approve
          </Button>
        ) : null}
        {actions.includes("submit") && canSubmit ? (
          <Button size="sm" variant="outline" disabled={busy} onClick={() => onDecide("Submitted")}>
            <SendIcon className="size-3.5" />
            Hand over
          </Button>
        ) : null}
      </div>
    </li>
  );
}

function TimesheetSheet({
  id,
  onOpenChange,
}: {
  id: string | null;
  onOpenChange: (open: boolean) => void;
}) {
  const detail = useQuery({
    queryKey: [TIMESHEET_KEY, id],
    queryFn: ({ signal }) => fetchTimesheet(id as string, { signal }),
    enabled: Boolean(id),
  });

  const sheet = detail.data;
  const tone = sheet ? timesheetStatusTone(sheet.status) : null;

  return (
    <Sheet open={Boolean(id)} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-lg">
        <SheetHeader>
          <SheetTitle className="flex items-center gap-2">
            {sheet?.worker ? `${sheet.worker.firstName} ${sheet.worker.lastName}` : "Timesheet"}
            {tone ? <Badge variant={tone.variant}>{tone.label}</Badge> : null}
          </SheetTitle>
          <SheetDescription>
            {sheet
              ? `Week of ${formatShiftDate(sheet.periodStart)} · overtime past ${formatHours(
                  sheet.overtimeThresholdMinutes,
                )}`
              : "Loading"}
          </SheetDescription>
        </SheetHeader>

        {detail.isLoading || !sheet ? (
          <div className="flex flex-col gap-3 px-4">
            <Skeleton className="h-20 w-full" />
            <Skeleton className="h-48 w-full" />
          </div>
        ) : (
          <div className="flex flex-col gap-4 overflow-y-auto px-4 pb-4">
            <div className="bg-muted/30 rounded-lg border p-3">
              <p className="font-mono text-2xl font-semibold tabular-nums">
                {formatHours(sheet.totalMinutes)}
              </p>
              <div className="mt-2">
                <SegmentedBar segments={hourSegments(sheet)} />
              </div>
              <dl className="mt-3 grid grid-cols-3 gap-2 text-xs">
                <Figure label="Regular" value={formatHours(sheet.regularMinutes)} color={REGULAR} />
                <Figure
                  label="Overtime"
                  value={formatHours(sheet.overtimeMinutes)}
                  color={OVERTIME}
                />
                <Figure
                  label="Paid leave"
                  value={formatHours(sheet.paidLeaveMinutes)}
                  color={LEAVE}
                />
              </dl>
            </div>

            {sheet.decisionNote ? (
              <p className="text-muted-foreground rounded-md border p-2 text-xs">
                {sheet.decisionNote}
              </p>
            ) : null}

            <section>
              <h4 className="text-xs font-semibold">Punches</h4>
              {(sheet.entries?.length ?? 0) === 0 ? (
                <p className="text-muted-foreground mt-1 text-xs">No punches on this week.</p>
              ) : (
                <ol className="mt-2 flex flex-col">
                  {(sheet.entries ?? []).map((entry) => (
                    <li
                      key={entry.id}
                      className="relative flex items-start gap-3 py-1.5 pl-4 text-xs before:absolute before:top-0 before:bottom-0 before:left-[5px] before:w-px before:bg-border"
                    >
                      <span
                        className="bg-background border-border absolute top-2.5 left-0 size-2.5 rounded-full border-2"
                        aria-hidden
                      />
                      <div className="flex min-w-0 flex-1 flex-col">
                        <span className="flex flex-wrap items-center gap-2">
                          <span className="font-medium tabular-nums">
                            {formatShiftDate(entry.clockedInAt)}
                          </span>
                          <span className="text-muted-foreground tabular-nums">
                            {formatPunchTime(entry.clockedInAt)}
                            {entry.clockedOutAt ? ` – ${formatPunchTime(entry.clockedOutAt)}` : ""}
                          </span>
                          {entry.source !== "Clock" ? (
                            <Badge variant="secondary">{entry.source}</Badge>
                          ) : null}
                        </span>
                        {entry.editReason ? (
                          <span className="text-muted-foreground">{entry.editReason}</span>
                        ) : null}
                      </div>
                      <span className="font-mono tabular-nums">
                        {formatHours(entry.paidMinutes)}
                        {entry.breakMinutes > 0 ? (
                          <span className="text-muted-foreground">
                            {" "}
                            · {entry.breakMinutes}m break
                          </span>
                        ) : null}
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </section>
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}

function Figure({ label, value, color }: { label: string; value: string; color: string }) {
  return (
    <div>
      <dt className="text-muted-foreground flex items-center gap-1.5 text-[11px]">
        <span className="size-1.5 rounded-full" style={{ background: color }} aria-hidden />
        {label}
      </dt>
      <dd className="font-mono font-semibold tabular-nums">{value}</dd>
    </div>
  );
}
