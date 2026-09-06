import { WorkerAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  clockIn,
  clockOut,
  fetchOpenTimeEntry,
  fetchTimeClockEntries,
  OPEN_ENTRY_KEY,
  TIMESHEETS_KEY,
  TIME_ENTRIES_KEY,
} from "@/lib/graphql/timesheet";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixInUserTimezone } from "@trenova/shared/lib/date";
import { formatShiftDate, startOfRotaWeek } from "@trenova/shared/lib/scheduling";
import { elapsedMinutes, formatHours } from "@trenova/shared/lib/timesheet";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  ClockIcon,
  PenLineIcon,
  PlayIcon,
  SquareIcon,
  Trash2Icon,
  UserRoundSearchIcon,
} from "lucide-react";
import { useEffect, useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { RecordEntryDialog, type EditableTimeEntry } from "./record-entry-dialog";
import { RemoveEntryDialog, type RemovableTimeEntry } from "./remove-entry-dialog";

type PickerValues = { workerId: string };

const SECONDS_IN_DAY = 86400;
const CLOCK_TICK_MS = 30_000;

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

/**
 * The clock. A worker is picked rather than inferred: a user account and a
 * worker record are different things in this system, and guessing at the link
 * would clock the wrong person in.
 */
export function TimeClockPanel() {
  const queryClient = useQueryClient();
  const { allowed: canRecord } = usePermission(Resource.Timesheet, Operation.Create);
  const { allowed: canCorrect } = usePermission(Resource.Timesheet, Operation.Update);
  const [entryDialog, setEntryDialog] = useState<{ entry: EditableTimeEntry | null } | null>(null);
  const [removing, setRemoving] = useState<RemovableTimeEntry | null>(null);
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  const form = useForm<PickerValues>({ defaultValues: { workerId: "" } });
  const workerId = useWatch({ control: form.control, name: "workerId" });

  // The running total is what somebody watches while they are on the clock, so
  // it ticks rather than waiting for the next request.
  useEffect(() => {
    const timer = setInterval(() => setNow(Math.floor(Date.now() / 1000)), CLOCK_TICK_MS);
    return () => clearInterval(timer);
  }, []);

  const openEntry = useQuery({
    queryKey: [OPEN_ENTRY_KEY, workerId],
    queryFn: ({ signal }) => fetchOpenTimeEntry(workerId, { signal }),
    enabled: Boolean(workerId),
  });

  // The window is anchored to the start of the week rather than to the ticking
  // clock: keying it on "now" would refetch the list every thirty seconds.
  const windowFrom = startOfRotaWeek(now) - SECONDS_IN_DAY * 7;
  const recentEntries = useQuery({
    queryKey: [TIME_ENTRIES_KEY, workerId, windowFrom],
    queryFn: ({ signal }) =>
      fetchTimeClockEntries({ workerId, from: windowFrom, limit: 50 }, { signal }),
    enabled: Boolean(workerId),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: [OPEN_ENTRY_KEY, workerId] });
    void queryClient.invalidateQueries({ queryKey: [TIME_ENTRIES_KEY, workerId] });
    void queryClient.invalidateQueries({ queryKey: [TIMESHEETS_KEY] });
  };

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
      toast.success("Clocked in");
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
      toast.success("Clocked out");
      invalidate();
    },
  });

  const entry = openEntry.data;
  const running = entry ? elapsedMinutes(entry.clockedInAt, now) : 0;
  const entries = recentEntries.data ?? [];
  const weekMinutes = entries
    .filter((row) => row.clockedInAt >= startOfRotaWeek(now))
    .reduce((sum, row) => sum + row.paidMinutes, 0);

  return (
    <div className="flex flex-col gap-4">
      <FormProvider {...form}>
        <Form onSubmit={(event) => event.preventDefault()}>
          <FormGroup cols={2}>
            <FormControl cols="full">
              <WorkerAutocompleteField<PickerValues>
                control={form.control}
                name="workerId"
                label="Worker"
                placeholder="Who is on the clock"
                clearable
                description="Only employees are paid by the clock — a contractor invoices instead."
              />
            </FormControl>
          </FormGroup>
        </Form>
      </FormProvider>

      {!workerId ? (
        <EmptyState
          className="max-w-none"
          title="Pick a worker"
          description="Their clock and the last two weeks of punches appear here."
          icons={[UserRoundSearchIcon, ClockIcon, PenLineIcon]}
        />
      ) : openEntry.isLoading ? (
        <Skeleton className="h-28 w-full rounded-xl" />
      ) : (
        <section
          className={cn(
            "border-border/80 bg-card relative flex flex-wrap items-center justify-between gap-4 overflow-hidden rounded-xl border p-5 transition-colors",
            entry && "border-emerald-500/40",
          )}
        >
          {entry ? (
            <span
              className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_top_left,rgb(16_185_129/0.10),transparent_60%)]"
              aria-hidden
            />
          ) : null}
          <div className="relative flex items-center gap-4">
            <span className="relative grid size-12 place-items-center rounded-full border">
              {entry ? (
                <>
                  <span className="absolute inset-0 animate-ping rounded-full bg-emerald-500/20" />
                  <span className="size-3 rounded-full bg-emerald-500" />
                </>
              ) : (
                <ClockIcon className="text-muted-foreground size-5" />
              )}
            </span>
            <div>
              <p className="text-muted-foreground text-[11px] font-medium tracking-wide uppercase">
                {entry ? "On the clock" : "Off the clock"}
              </p>
              <p className="font-mono text-2xl leading-none font-semibold tabular-nums">
                {entry ? formatHours(running) : formatHours(weekMinutes)}
              </p>
              <p className="text-muted-foreground mt-1 text-xs">
                {entry
                  ? `Since ${formatPunchInstant(entry.clockedInAt)}`
                  : `${formatHours(weekMinutes)} paid this week so far`}
              </p>
            </div>
          </div>

          <div className="relative flex items-center gap-2">
            {canCorrect ? (
              <Button size="sm" variant="outline" onClick={() => setEntryDialog({ entry: null })}>
                <PenLineIcon className="size-3.5" />
                Record hours
              </Button>
            ) : null}
            {canRecord ? (
              entry ? (
                <Button
                  size="sm"
                  variant="destructive"
                  isLoading={clockingOut}
                  onClick={() => void punchOut({ workerId })}
                >
                  <SquareIcon className="size-3.5" />
                  Clock out
                </Button>
              ) : (
                <Button size="sm" isLoading={clockingIn} onClick={() => void punchIn({ workerId })}>
                  <PlayIcon className="size-3.5" />
                  Clock in
                </Button>
              )
            ) : null}
          </div>
        </section>
      )}

      {workerId ? (
        <section className="border-border/80 bg-card rounded-xl border p-4">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-semibold">Last two weeks</h3>
            <span className="text-muted-foreground text-xs tabular-nums">
              {entries.length} punch{entries.length === 1 ? "" : "es"}
            </span>
          </div>
          {recentEntries.isLoading ? (
            <Skeleton className="mt-3 h-32 w-full" />
          ) : entries.length === 0 ? (
            <p className="text-muted-foreground mt-2 text-xs">No punches recorded.</p>
          ) : (
            <ul className="mt-3 flex flex-col">
              {[...entries].reverse().map((row) => (
                <li
                  key={row.id}
                  className="group/row hover:bg-muted/40 -mx-2 flex flex-wrap items-center justify-between gap-2 rounded-md px-2 py-1.5 text-xs transition-colors"
                >
                  <span className="flex min-w-0 flex-wrap items-center gap-2">
                    <span
                      className={cn(
                        "size-1.5 shrink-0 rounded-full",
                        row.clockedOutAt ? "bg-muted-foreground/50" : "bg-emerald-500",
                      )}
                      aria-hidden
                    />
                    <span className="font-medium tabular-nums">
                      {formatShiftDate(row.clockedInAt)}
                    </span>
                    <span className="text-muted-foreground tabular-nums">
                      {formatPunchTime(row.clockedInAt)}
                      {row.clockedOutAt ? ` – ${formatPunchTime(row.clockedOutAt)}` : ""}
                    </span>
                    {row.source !== "Clock" ? (
                      <Badge variant="secondary">{row.source}</Badge>
                    ) : null}
                    {row.editReason ? (
                      <span className="text-muted-foreground truncate">· {row.editReason}</span>
                    ) : null}
                  </span>
                  <span className="flex items-center gap-1">
                    <span className="font-mono tabular-nums">
                      {row.clockedOutAt ? formatHours(row.paidMinutes) : "Running"}
                      {row.breakMinutes > 0 ? (
                        <span className="text-muted-foreground"> · {row.breakMinutes}m break</span>
                      ) : null}
                    </span>
                    {canCorrect && row.clockedOutAt ? (
                      <span className="flex items-center opacity-0 transition-opacity group-hover/row:opacity-100 focus-within:opacity-100">
                        <Button
                          size="icon-xs"
                          variant="ghost"
                          aria-label="Correct this entry"
                          onClick={() => setEntryDialog({ entry: row })}
                        >
                          <PenLineIcon className="size-3.5" />
                        </Button>
                        <Button
                          size="icon-xs"
                          variant="ghost"
                          className="text-destructive hover:text-destructive"
                          aria-label="Remove this entry"
                          onClick={() => setRemoving(row)}
                        >
                          <Trash2Icon className="size-3.5" />
                        </Button>
                      </span>
                    ) : null}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>
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
