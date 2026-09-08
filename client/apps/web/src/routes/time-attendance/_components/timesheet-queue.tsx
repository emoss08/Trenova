import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchTimesheet,
  TIMESHEETS_KEY,
  TIMESHEET_KEY,
  transitionTimesheet,
  type TimesheetDetail,
  type TimesheetRow,
} from "@/lib/graphql/timesheet";
import {
  countBySegment,
  sheetsInSegment,
  waitingDays,
  weekCardDays,
  type QueueSegment,
} from "@/lib/time-attendance";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, AvatarFallback } from "@trenova/shared/components/ui/avatar";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { CompositionBar } from "@trenova/shared/components/ui/composition-bar";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import {
  SegmentedControl,
  type SegmentedControlItem,
} from "@trenova/shared/components/ui/segmented-control";
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
import { cn, initials } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { CheckIcon, SendIcon, UndoIcon, UsersIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useEffect, useMemo, useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { activeQueueQuery, paidQueueQuery } from "./queries";
import { TimesheetsEmpty } from "./time-attendance-empty";

type FilterValues = { workerId: string };

const STAGGER_LIMIT = 10;

const SEGMENT_LABELS: Record<QueueSegment, string> = {
  Submitted: "Awaiting approval",
  Open: "Open",
  Approved: "Approved",
  Locked: "Paid",
};

const EMPTY_TITLES: Record<QueueSegment, string> = {
  Submitted: "Nothing awaiting approval",
  Open: "No open weeks",
  Approved: "Nothing approved",
  Locked: "Nothing paid yet",
};

const EMPTY_DESCRIPTIONS: Record<QueueSegment, string> = {
  Submitted: "A week somebody hands over lands here, with the hours frozen as they were.",
  Open: "Weeks still being worked, and any sent back for another look, land here.",
  Approved: "A week a manager has signed off waits here until payroll picks it up.",
  Locked: "A week goes here once a payroll run has carried it.",
};

function hourSegments(sheet: {
  regularMinutes: number;
  overtimeMinutes: number;
  paidLeaveMinutes: number;
}) {
  return [
    { key: "regular", label: "Regular", value: sheet.regularMinutes },
    { key: "overtime", label: "Overtime", value: sheet.overtimeMinutes },
    { key: "leave", label: "Paid leave", value: sheet.paidLeaveMinutes },
  ];
}

function formatPunchTime(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, { hour: "numeric", minute: "2-digit" });
}

function formatCardDay(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, {
    weekday: "short",
    month: "short",
    day: "numeric",
    timezone: "UTC",
  });
}

function sheetWorkerName(sheet: {
  workerId: string;
  worker?: { firstName: string; lastName: string } | null;
}): string {
  return sheet.worker ? `${sheet.worker.firstName} ${sheet.worker.lastName}` : sheet.workerId;
}

/**
 * The approval queue. Submitted weeks come first because they are the ones
 * waiting on somebody, and a week's totals are what was frozen at submit:
 * approving it is approving those numbers, not whatever the punches say now.
 */
export function TimesheetQueue({ now }: { now: number }) {
  const queryClient = useQueryClient();
  const { allowed: canApprove } = usePermission(Resource.Timesheet, Operation.Approve);
  const { allowed: canSubmit } = usePermission(Resource.Timesheet, Operation.Submit);
  const [segment, setSegment] = useState<QueueSegment>("Submitted");
  const [teamOnly, setTeamOnly] = useState(false);
  const [openSheetId, setOpenSheetId] = useState<string | null>(null);

  const filterForm = useForm<FilterValues>({ defaultValues: { workerId: "" } });
  const workerId = useWatch({ control: filterForm.control, name: "workerId" });

  // The three live segments come from one request so their counts are on the
  // control before a segment is opened; paid weeks are history and load on
  // their own when asked for.
  const filter = { teamOnly, workerId: workerId || null };
  const active = useQuery(activeQueueQuery(filter));
  const paid = useQuery({ ...paidQueueQuery(filter), enabled: segment === "Locked" });

  const counts = useMemo(() => countBySegment(active.data ?? []), [active.data]);
  const segmentItems = useMemo<SegmentedControlItem<QueueSegment>[]>(
    () => [
      { value: "Submitted", label: SEGMENT_LABELS.Submitted, caption: String(counts.Submitted) },
      { value: "Open", label: SEGMENT_LABELS.Open, caption: String(counts.Open) },
      { value: "Approved", label: SEGMENT_LABELS.Approved, caption: String(counts.Approved) },
      { value: "Locked", label: SEGMENT_LABELS.Locked },
    ],
    [counts],
  );

  const sheets = segment === "Locked" ? paid : active;
  const rows = useMemo(
    () => (segment === "Locked" ? (paid.data ?? []) : sheetsInSegment(active.data ?? [], segment)),
    [segment, paid.data, active.data],
  );

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

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <SegmentedControl<QueueSegment>
          items={segmentItems}
          value={segment}
          onValueChange={setSegment}
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
                    description="Shows only this person's weeks; clear it to see everyone."
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
        <TimesheetsEmpty
          title={EMPTY_TITLES[segment]}
          description={
            workerId || teamOnly
              ? "Nothing here for that filter. Clear it to see everyone's weeks."
              : EMPTY_DESCRIPTIONS[segment]
          }
        />
      ) : (
        <QueueList
          key={segment}
          rows={rows}
          segment={segment}
          now={now}
          busy={isPending}
          canApprove={canApprove}
          canSubmit={canSubmit}
          onOpen={setOpenSheetId}
          onDecide={(id, next) => decide({ id, status: next })}
        />
      )}

      <TimesheetSheet id={openSheetId} onOpenChange={(open) => !open && setOpenSheetId(null)} />
    </div>
  );
}

type QueueListProps = {
  rows: readonly TimesheetRow[];
  segment: QueueSegment;
  now: number;
  busy: boolean;
  canApprove: boolean;
  canSubmit: boolean;
  onOpen: (id: string) => void;
  onDecide: (id: string, status: "Submitted" | "Approved" | "Rejected") => void;
};

function QueueList({
  rows,
  segment,
  now,
  busy,
  canApprove,
  canSubmit,
  onOpen,
  onDecide,
}: QueueListProps) {
  const reduceMotion = useReducedMotion();
  const [settled, setSettled] = useState(false);

  useEffect(() => {
    const timer = window.setTimeout(() => setSettled(true), 600);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <ul
      className="bg-card divide-y overflow-hidden rounded-lg border"
      aria-label={SEGMENT_LABELS[segment]}
    >
      {rows.map((sheet, index) => (
        <m.li
          key={sheet.id}
          initial={settled || reduceMotion ? false : { opacity: 0, y: 4 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.22, delay: Math.min(index, STAGGER_LIMIT) * 0.03 }}
        >
          <QueueRow
            sheet={sheet}
            now={now}
            busy={busy}
            canApprove={canApprove}
            canSubmit={canSubmit}
            onOpen={() => onOpen(sheet.id)}
            onDecide={(next) => onDecide(sheet.id, next)}
          />
        </m.li>
      ))}
    </ul>
  );
}

function QueueRow({
  sheet,
  now,
  busy,
  canApprove,
  canSubmit,
  onOpen,
  onDecide,
}: {
  sheet: TimesheetRow;
  now: number;
  busy: boolean;
  canApprove: boolean;
  canSubmit: boolean;
  onOpen: () => void;
  onDecide: (status: "Submitted" | "Approved" | "Rejected") => void;
}) {
  const tone = timesheetStatusTone(sheet.status);
  const actions = timesheetActionsFor(sheet.status, { isOwner: false, canApprove });
  const name = sheetWorkerName(sheet);
  const waited =
    sheet.status === "Submitted" && sheet.submittedAt ? waitingDays(sheet.submittedAt, now) : null;

  const meta = [
    `${sheet.entryCount} punch${sheet.entryCount === 1 ? "" : "es"}`,
    sheet.status !== "Submitted" && sheet.submittedAt
      ? `handed over ${formatShiftDate(sheet.submittedAt)}`
      : null,
    sheet.approvedAt ? `approved ${formatShiftDate(sheet.approvedAt)}` : null,
    sheet.status === "Rejected" && sheet.decisionNote ? sheet.decisionNote : null,
  ].filter(Boolean);

  return (
    <div className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-4 px-3 py-2.5 text-xs md:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)_auto]">
      <button
        type="button"
        className="focus-visible:ring-ring/60 flex min-w-0 items-center gap-3 rounded-md text-left outline-none focus-visible:ring-2"
        onClick={onOpen}
      >
        <Avatar size="sm">
          <AvatarFallback className="text-2xs font-medium">
            {sheet.worker ? initials(sheet.worker.firstName, sheet.worker.lastName) : "?"}
          </AvatarFallback>
        </Avatar>
        <span className="flex min-w-0 flex-1 flex-col gap-0.5 leading-tight">
          <span className="flex min-w-0 flex-wrap items-center gap-2">
            <span className="truncate text-sm font-medium">{name}</span>
            <span className="text-muted-foreground tabular-nums">
              Week of {formatShiftDate(sheet.periodStart)}
            </span>
            <Badge variant={tone.variant}>{tone.label}</Badge>
          </span>
          <span className="text-muted-foreground flex min-w-0 flex-wrap items-center gap-x-2">
            {waited !== null ? (
              <span className={cn("tabular-nums", waited >= 3 && "text-warning-foreground")}>
                {waited === 0
                  ? "Handed over today"
                  : `Waiting ${waited} day${waited === 1 ? "" : "s"}`}
              </span>
            ) : null}
            {meta.length > 0 ? <span className="truncate">{meta.join(" · ")}</span> : null}
          </span>
        </span>
      </button>

      <div className="col-span-2 flex min-w-0 items-center gap-3 md:col-span-1">
        <CompositionBar
          size="sm"
          showLegend={false}
          className="min-w-0 flex-1"
          aria-label={`${name}'s hours`}
          formatValue={formatHours}
          segments={hourSegments(sheet)}
        />
        <span className="text-right leading-tight">
          <span className="block font-mono text-sm font-semibold tabular-nums">
            {formatHours(sheet.totalMinutes)}
          </span>
          {sheet.overtimeMinutes > 0 ? (
            <span className="text-muted-foreground block tabular-nums">
              {formatHours(sheet.overtimeMinutes)} overtime
            </span>
          ) : null}
        </span>
      </div>

      {actions.length > 0 ? (
        <div className="flex shrink-0 items-center gap-1.5 md:col-start-3">
          {actions.includes("reject") ? (
            <Button
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => onDecide("Rejected")}
            >
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
            <Button
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => onDecide("Submitted")}
            >
              <SendIcon className="size-3.5" />
              Hand over
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
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
        <SheetHeader className="pr-10">
          <SheetTitle className="flex items-center gap-2">
            {sheet ? sheetWorkerName(sheet) : "Timesheet"}
            {tone ? <Badge variant={tone.variant}>{tone.label}</Badge> : null}
          </SheetTitle>
          <SheetDescription>
            {sheet
              ? `Week of ${formatShiftDate(sheet.periodStart)}, overtime past ${formatHours(
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
          <div className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-4 pb-4">
            <div className="flex flex-col gap-2">
              <div className="flex items-baseline justify-between gap-3">
                <span className="font-mono text-2xl leading-none font-semibold tabular-nums">
                  {formatHours(sheet.totalMinutes)}
                </span>
                <span className="text-muted-foreground text-xs tabular-nums">
                  {sheet.entryCount} punch{sheet.entryCount === 1 ? "" : "es"}
                </span>
              </div>
              <CompositionBar
                aria-label="Hours by kind"
                formatValue={formatHours}
                segments={hourSegments(sheet)}
              />
            </div>

            <TimeCard sheet={sheet} />

            {sheet.decisionNote ? (
              <p className="bg-muted/60 text-muted-foreground rounded-md px-3 py-2 text-xs">
                {sheet.decisionNote}
              </p>
            ) : null}

            <section aria-label="Punches" className="flex flex-col gap-1.5">
              <h4 className="text-muted-foreground text-xs font-medium">Punches</h4>
              {(sheet.entries?.length ?? 0) === 0 ? (
                <p className="text-muted-foreground text-xs">No punches on this week.</p>
              ) : (
                <ol className="flex flex-col">
                  {(sheet.entries ?? []).map((entry) => (
                    <li
                      key={entry.id}
                      className="before:bg-border relative flex items-start gap-3 py-1.5 pl-4 text-xs before:absolute before:top-0 before:bottom-0 before:left-[5px] before:w-px"
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

/**
 * The week as a time card: seven cells, one per rota day, each carrying the
 * hours punched on it. The bars are relative to the longest day of the week,
 * so an uneven week shows its shape before its numbers are read.
 */
function TimeCard({ sheet }: { sheet: TimesheetDetail }) {
  const days = useMemo(
    () => weekCardDays(sheet.entries ?? [], sheet.periodStart, sheet.periodEnd),
    [sheet.entries, sheet.periodStart, sheet.periodEnd],
  );
  const peak = Math.max(1, ...days.map((day) => day.paidMinutes));
  const reduceMotion = useReducedMotion();

  return (
    <div role="group" aria-label="Time card" className="grid grid-cols-7 gap-1">
      {days.map((day, index) => {
        const label = formatCardDay(day.startsAt);
        return (
          <div
            key={day.startsAt}
            role="group"
            aria-label={label}
            className={cn(
              "flex flex-col items-center gap-1.5 rounded-md border px-1 py-2",
              day.punches === 0 ? "border-border/50 border-dashed" : "border-border/80 bg-card",
            )}
          >
            <span className="text-muted-foreground text-2xs leading-none">{label.slice(0, 3)}</span>
            <span className="bg-muted relative h-12 w-2 overflow-hidden rounded-full">
              <m.span
                initial={reduceMotion ? false : { scaleY: 0 }}
                animate={{ scaleY: 1 }}
                transition={{ duration: 0.4, delay: index * 0.04, ease: "easeOut" }}
                className={cn(
                  "absolute inset-x-0 bottom-0 origin-bottom rounded-full",
                  day.running ? "bg-success/80" : "bg-brand/70",
                )}
                style={{ height: `${(day.paidMinutes / peak) * 100}%` }}
              />
            </span>
            <span
              className={cn(
                "font-mono text-2xs leading-none font-medium tabular-nums",
                day.punches === 0 && "text-muted-foreground/60",
              )}
            >
              {day.punches === 0
                ? "—"
                : day.running
                  ? `${formatHours(day.paidMinutes)}+`
                  : formatHours(day.paidMinutes)}
            </span>
          </div>
        );
      })}
    </div>
  );
}
