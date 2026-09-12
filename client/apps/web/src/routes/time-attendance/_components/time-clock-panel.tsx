import { useT } from "@trenova/shared/i18n/use-t";
import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  clockIn,
  clockOut,
  fetchOpenTimeEntry,
  fetchTimeClockEntries,
  OPEN_ENTRIES_KEY,
  OPEN_ENTRY_KEY,
  TIME_ENTRIES_KEY,
  TIMESHEETS_KEY,
  type OpenTimeEntryRow,
  type TimeClockEntryRow,
} from "@/lib/graphql/timesheet";
import {
  dayTrackSpans,
  groupEntriesByDay,
  overtimeHeadroom,
  type EntryDay,
} from "@/lib/time-attendance";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixInUserTimezone, resolveUserTimezone } from "@trenova/shared/lib/date";
import { startOfRotaWeek } from "@trenova/shared/lib/scheduling";
import { elapsedMinutes, formatHours, timesheetStatusTone } from "@trenova/shared/lib/timesheet";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ClockIcon, PenLineIcon, PlayIcon, SquareIcon, Trash2Icon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import { useMemo, useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { ClockBoard } from "./clock-board";
import { workerWeekQuery } from "./queries";
import { RecordEntryDialog, type EditableTimeEntry } from "./record-entry-dialog";
import { RemoveEntryDialog, type RemovableTimeEntry } from "./remove-entry-dialog";

type PickerValues = { workerId: string };

const SECONDS_IN_DAY = 86400;
const EMPTY_ENTRIES: TimeClockEntryRow[] = [];
const TRACK_TICKS = [6, 12, 18];

// A punch is an instant, so it renders in the reader's own zone. The rota is
// keyed on UTC days and renders in UTC; the two are different questions.
function formatPunchInstant(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  });
}

function formatPunchTime(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, { hour: "numeric", minute: "2-digit" });
}

function formatDayHeading(unixSeconds: number): string {
  return formatUnixInUserTimezone(unixSeconds, {
    weekday: "short",
    month: "short",
    day: "numeric",
  });
}

type TimeClockPanelProps = {
  now: number;
  openEntries: readonly OpenTimeEntryRow[] | undefined;
  openEntriesLoading: boolean;
  teamOnly: boolean;
  onTeamOnlyChange: (value: boolean) => void;
};

/**
 * The clock. A worker is picked rather than inferred: a user account and a
 * worker record are different things in this system, and guessing at the link
 * would clock the wrong person in. The board above the picker is the shortcut:
 * anybody already punched in is one click from their clock.
 */
export function TimeClockPanel({
  now,
  openEntries,
  openEntriesLoading,
  teamOnly,
  onTeamOnlyChange,
}: TimeClockPanelProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const { allowed: canRecord } = usePermission(Resource.Timesheet, Operation.Create);
  const { allowed: canCorrect } = usePermission(Resource.Timesheet, Operation.Update);
  const [entryDialog, setEntryDialog] = useState<{ entry: EditableTimeEntry | null } | null>(null);
  const [removing, setRemoving] = useState<RemovableTimeEntry | null>(null);

  const form = useForm<PickerValues>({ defaultValues: { workerId: "" } });
  const workerId = useWatch({ control: form.control, name: "workerId" });

  const openEntry = useQuery({
    queryKey: [OPEN_ENTRY_KEY, workerId],
    queryFn: ({ signal }) => fetchOpenTimeEntry(workerId, { signal }),
    enabled: Boolean(workerId),
  });

  // The window is anchored to the start of the week rather than to the ticking
  // clock: keying it on "now" would refetch the list every thirty seconds.
  const weekStart = startOfRotaWeek(now);
  const windowFrom = weekStart - SECONDS_IN_DAY * 7;
  const recentEntries = useQuery({
    queryKey: [TIME_ENTRIES_KEY, workerId, windowFrom],
    queryFn: ({ signal }) =>
      fetchTimeClockEntries({ workerId, from: windowFrom, limit: 50 }, { signal }),
    enabled: Boolean(workerId),
  });
  const workerWeek = useQuery({
    ...workerWeekQuery(workerId, weekStart),
    enabled: Boolean(workerId),
  });

  const invalidateWorker = (id: string) => {
    void queryClient.invalidateQueries({ queryKey: [OPEN_ENTRY_KEY, id] });
    void queryClient.invalidateQueries({ queryKey: [TIME_ENTRIES_KEY, id] });
    void queryClient.invalidateQueries({ queryKey: [OPEN_ENTRIES_KEY] });
    void queryClient.invalidateQueries({ queryKey: [TIMESHEETS_KEY] });
  };
  const invalidate = () => invalidateWorker(workerId);

  const { mutateAsync: punchIn, isPending: clockingIn } = useApiMutation<
    { id: string },
    PickerValues,
    unknown,
    PickerValues
  >({
    form,
    resourceName: "Punch",
    mutationFn: () => clockIn({ workerId }),
    onSuccess: () => {
      toast.success(t("Clocked in"));
      invalidate();
    },
  });

  const { mutateAsync: punchOut, isPending: clockingOut } = useApiMutation<
    { id: string },
    PickerValues,
    unknown,
    PickerValues
  >({
    form,
    resourceName: "Punch",
    mutationFn: () => clockOut({ workerId }),
    onSuccess: () => {
      toast.success(t("Clocked out"));
      invalidate();
    },
  });

  // Punching somebody out from the board does not go through the picker form:
  // the row names the worker, and the picker may be on somebody else.
  const boardPunchOut = useMutation({
    mutationFn: (id: string) => clockOut({ workerId: id }),
    onSuccess: (_data, id) => {
      toast.success(t("Clocked out"));
      invalidateWorker(id);
    },
    onError: (error: Error) =>
      toast.error(t("Could not clock out"), { description: error.message }),
  });

  const entry = openEntry.data;
  const running = entry ? elapsedMinutes(entry.clockedInAt, now) : 0;
  const entries = recentEntries.data ?? EMPTY_ENTRIES;
  const punchedMinutes = useMemo(
    () =>
      entries
        .filter((row) => row.clockedInAt >= weekStart)
        .reduce((sum, row) => sum + row.paidMinutes, 0),
    [entries, weekStart],
  );
  const weekSheet = useMemo(
    () => (workerWeek.data ?? []).find((row) => row.workerId === workerId) ?? null,
    [workerWeek.data, workerId],
  );
  // The sheet is the figure payroll will read; the punches are the fallback
  // for a week no sheet has been opened for yet.
  const weekMinutes = weekSheet ? weekSheet.totalMinutes : punchedMinutes;
  const headroom = weekSheet
    ? overtimeHeadroom(weekMinutes, weekSheet.overtimeThresholdMinutes)
    : null;
  const timezone = resolveUserTimezone();
  const days = useMemo(() => groupEntriesByDay(entries, timezone), [entries, timezone]);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="min-w-0">
          <h3 className="text-sm font-medium">{t("Clock")}</h3>
          <p className="text-muted-foreground mt-0.5 text-xs">
            {t(
              "Pick a worker, or choose somebody from the board, to work their clock and see the last two weeks of punches.",
            )}
          </p>
        </div>
        <div className="w-full sm:w-72">
          <FormProvider {...form}>
            <Form onSubmit={(event) => event.preventDefault()}>
              <FormGroup cols={1}>
                <FormControl cols="full">
                  <WorkerAutocompleteField<PickerValues>
                    control={form.control}
                    name="workerId"
                    label={t("Worker")}
                    placeholder={t("Who is on the clock")}
                    clearable
                  />
                </FormControl>
              </FormGroup>
            </Form>
          </FormProvider>
        </div>
      </div>

      <ClockBoard
        entries={openEntries}
        isLoading={openEntriesLoading}
        now={now}
        teamOnly={teamOnly}
        onTeamOnlyChange={onTeamOnlyChange}
        canPunch={canRecord}
        punchingOut={boardPunchOut.isPending ? boardPunchOut.variables : null}
        onPunchOut={(id) => boardPunchOut.mutate(id)}
        onPick={(id) => form.setValue("workerId", id, { shouldDirty: true })}
        selectedWorkerId={workerId}
      />

      {workerId ? (
        <div className="grid gap-4 xl:grid-cols-[22rem_minmax(0,1fr)]">
          <div className="bg-card flex flex-col self-start overflow-hidden rounded-lg border">
            {openEntry.isLoading ? (
              <div className="p-4">
                <Skeleton className="h-36 w-full rounded-lg" />
              </div>
            ) : (
              <section aria-label={t("Clock")} className="flex flex-col gap-4 p-4">
                <div className="flex items-center gap-4">
                  <span
                    className={cn(
                      "relative grid size-11 shrink-0 place-items-center rounded-full border",
                      entry && "border-success/40",
                    )}
                  >
                    {entry ? (
                      <>
                        <span className="bg-success/20 absolute inset-0 animate-ping rounded-full motion-reduce:hidden" />
                        <span className="bg-success size-2.5 rounded-full" />
                      </>
                    ) : (
                      <ClockIcon className="text-muted-foreground size-5" />
                    )}
                  </span>
                  <div className="min-w-0">
                    <p className="cc-label">{entry ? t("On the clock") : t("Off the clock")}</p>
                    <p className="font-mono text-2xl leading-none font-semibold tabular-nums">
                      {entry ? formatHours(running) : formatHours(weekMinutes)}
                    </p>
                    <p className="text-muted-foreground mt-1 text-xs">
                      {entry
                        ? t("Since {0}", formatPunchInstant(entry.clockedInAt))
                        : t("{0} paid this week so far", formatHours(weekMinutes))}
                    </p>
                  </div>
                </div>

                <div className="flex flex-col gap-1.5">
                  <div className="flex items-center justify-between gap-2 text-xs">
                    <span className="flex items-center gap-1.5">
                      <span className="font-medium">{t("This week")}</span>
                      {weekSheet ? (
                        <Badge variant={timesheetStatusTone(weekSheet.status).variant}>
                          {t(timesheetStatusTone(weekSheet.status).label)}
                        </Badge>
                      ) : null}
                    </span>
                    <span className="text-muted-foreground tabular-nums">
                      {headroom
                        ? t(
                            "{0} of {1}",
                            formatHours(weekMinutes),
                            formatHours(weekSheet!.overtimeThresholdMinutes),
                          )
                        : formatHours(weekMinutes)}
                    </span>
                  </div>
                  {headroom ? (
                    <>
                      <span
                        role="meter"
                        aria-label={t("This week")}
                        aria-valuemin={0}
                        aria-valuemax={100}
                        aria-valuenow={Math.round(headroom.share * 100)}
                        className="bg-muted relative h-1.5 overflow-hidden rounded-full"
                      >
                        <span
                          className={cn(
                            "absolute inset-y-0 left-0 rounded-full transition-[width] duration-700 ease-out motion-reduce:transition-none",
                            headroom.over > 0 ? "bg-warning" : "bg-brand",
                          )}
                          style={{ width: `${headroom.share * 100}%` }}
                        />
                      </span>
                      <p className="text-muted-foreground text-xs tabular-nums">
                        {headroom.over > 0
                          ? t("{0} into overtime", formatHours(headroom.over))
                          : t("{0} before overtime", formatHours(headroom.remaining))}
                      </p>
                    </>
                  ) : (
                    <p className="text-muted-foreground text-xs">
                      {t("No timesheet has been opened for this week yet.")}
                    </p>
                  )}
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  {canRecord ? (
                    entry ? (
                      <Button
                        size="sm"
                        variant="outline"
                        isLoading={clockingOut}
                        onClick={() => void punchOut({ workerId })}
                      >
                        <SquareIcon className="size-3.5" />
                        {t("Clock out")}
                      </Button>
                    ) : (
                      <Button
                        size="sm"
                        isLoading={clockingIn}
                        onClick={() => void punchIn({ workerId })}
                      >
                        <PlayIcon className="size-3.5" />
                        {t("Clock in")}
                      </Button>
                    )
                  ) : null}
                  {canCorrect ? (
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => setEntryDialog({ entry: null })}
                    >
                      <PenLineIcon className="size-3.5" />
                      {t("Record hours")}
                    </Button>
                  ) : null}
                </div>
              </section>
            )}
          </div>

          <section
            aria-labelledby="punch-history-heading"
            className="bg-card overflow-hidden rounded-lg border"
          >
            <header className="flex items-center justify-between gap-2 border-b px-3 py-2">
              <h3 id="punch-history-heading" className="text-sm font-medium">
                {t("Last two weeks")}
              </h3>
              <span className="text-muted-foreground text-xs tabular-nums">
                {t(
                  "{0, plural, one {# punch} other {# punches}} · {1}",
                  entries.length,
                  formatHours(entries.reduce((sum, row) => sum + row.paidMinutes, 0)),
                )}
              </span>
            </header>
            {recentEntries.isLoading ? (
              <div className="flex flex-col gap-2 p-3">
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-full" />
                <Skeleton className="h-8 w-3/4" />
              </div>
            ) : days.length === 0 ? (
              <p className="text-muted-foreground px-3 py-3 text-sm">
                {t("No punches in the last two weeks.")}
              </p>
            ) : (
              <div className="divide-y">
                {days.map((day) => (
                  <DayGroup
                    key={day.key}
                    day={day}
                    now={now}
                    timezone={timezone}
                    canCorrect={canCorrect}
                    onEdit={(row) => setEntryDialog({ entry: row })}
                    onRemove={setRemoving}
                  />
                ))}
              </div>
            )}
          </section>
        </div>
      ) : null}

      <RecordEntryDialog
        open={entryDialog !== null}
        onOpenChange={(open) => !open && setEntryDialog(null)}
        workerId={workerId}
        entry={entryDialog?.entry ?? null}
        onRecorded={invalidate}
      />
      <RemoveEntryDialog
        entry={removing}
        onOpenChange={(open) => !open && setRemoving(null)}
        onRemoved={invalidate}
      />
    </div>
  );
}

type DayGroupProps = {
  day: EntryDay<TimeClockEntryRow>;
  now: number;
  timezone: string;
  canCorrect: boolean;
  onEdit: (row: TimeClockEntryRow) => void;
  onRemove: (row: TimeClockEntryRow) => void;
};

/**
 * One day of punches: the heading with the day's total, the punches drawn
 * on a 24-hour track so a gap or a forgotten punch-out is visible at a glance,
 * then the punches themselves.
 */
function DayGroup({ day, now, timezone, canCorrect, onEdit, onRemove }: DayGroupProps) {
  const t = useT();

  const heading = formatDayHeading(day.startsAt);
  const spans = useMemo(
    () => dayTrackSpans(day.entries, now, timezone),
    [day.entries, now, timezone],
  );
  const reduceMotion = useReducedMotion();
  const trackName = day.entries
    .map((row) =>
      row.clockedOutAt
        ? `${formatPunchTime(row.clockedInAt)} to ${formatPunchTime(row.clockedOutAt)}`
        : `${formatPunchTime(row.clockedInAt)} still running`,
    )
    .join(", ");

  return (
    <section aria-label={heading} className="flex flex-col">
      <header className="flex items-center justify-between gap-2 px-3 pt-2.5 pb-1">
        <h4 className="text-xs font-medium">{heading}</h4>
        <span className="text-xs tabular-nums" aria-label={t("Day total")}>
          <span className="font-mono font-medium">
            {day.running
              ? t("{0} + running", formatHours(day.paidMinutes))
              : formatHours(day.paidMinutes)}
          </span>
          {day.breakMinutes > 0 ? (
            <span className="text-muted-foreground"> {t("· {0}m break", day.breakMinutes)}</span>
          ) : null}
        </span>
      </header>
      <div className="px-3 pb-2">
        <div
          role="img"
          aria-label={`Punches on ${heading}: ${trackName}`}
          className="bg-muted relative h-2 w-full overflow-hidden rounded-full"
        >
          {TRACK_TICKS.map((hour) => (
            <span
              key={hour}
              aria-hidden
              className="bg-background absolute inset-y-0 w-px"
              style={{ left: `${(hour / 24) * 100}%` }}
            />
          ))}
          {spans.map((span, index) => (
            <m.span
              key={span.id}
              data-slot="punch-span"
              initial={reduceMotion ? false : { scaleX: 0 }}
              animate={{ scaleX: 1 }}
              transition={{ duration: 0.4, delay: index * 0.05, ease: "easeOut" }}
              className={cn(
                "absolute inset-y-0 origin-left rounded-full",
                span.running ? "bg-success/80" : "bg-brand/70",
              )}
              style={{
                left: `${span.start * 100}%`,
                width: `${(span.end - span.start) * 100}%`,
              }}
            />
          ))}
        </div>
      </div>
      <ul className="divide-y">
        {day.entries.map((row) => (
          <li
            key={row.id}
            className="group/row grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-3 py-2 text-xs"
          >
            <span className="flex min-w-0 flex-wrap items-center gap-2">
              <span
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  row.clockedOutAt ? "bg-brand/70" : "bg-success",
                )}
                aria-hidden
              />
              <span className="font-medium tabular-nums">
                {formatPunchTime(row.clockedInAt)}
                {row.clockedOutAt ? ` – ${formatPunchTime(row.clockedOutAt)}` : ` ${t("– now")}`}
              </span>
              {row.source !== "Clock" ? <Badge variant="secondary">{row.source}</Badge> : null}
              {row.editReason ? (
                <span className="text-muted-foreground truncate">· {row.editReason}</span>
              ) : null}
            </span>
            <span className="flex items-center gap-1">
              <span className="font-mono tabular-nums">
                {row.clockedOutAt ? formatHours(row.paidMinutes) : t("Running")}
                {row.breakMinutes > 0 ? (
                  <span className="text-muted-foreground">
                    {" "}
                    {t("· {0}m break", row.breakMinutes)}
                  </span>
                ) : null}
              </span>
              {canCorrect && row.clockedOutAt ? (
                <span className="flex items-center opacity-0 transition-opacity group-hover/row:opacity-100 focus-within:opacity-100">
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    aria-label={t("Correct this entry")}
                    onClick={() => onEdit(row)}
                  >
                    <PenLineIcon className="size-3.5" />
                  </Button>
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    className="text-destructive hover:text-destructive"
                    aria-label={t("Remove this entry")}
                    onClick={() => onRemove(row)}
                  >
                    <Trash2Icon className="size-3.5" />
                  </Button>
                </span>
              ) : null}
            </span>
          </li>
        ))}
      </ul>
    </section>
  );
}
