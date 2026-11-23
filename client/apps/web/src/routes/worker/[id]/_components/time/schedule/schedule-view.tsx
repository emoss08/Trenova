import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { VirtualRows } from "@/components/virtual-rows";
import { lazyWorkPane, preloadOn, useWorkPane } from "@/components/work-pane";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import {
  AVAILABILITY_PREFERENCES_KEY,
  endWorkerShiftAssignment,
  fetchShiftSwapRequests,
  fetchWorkerAvailabilityPreferences,
  fetchWorkerShiftAssignments,
  setWorkerAvailabilityPreference,
  SHIFT_ASSIGNMENTS_KEY,
  SHIFT_SWAPS_KEY,
  type ShiftAssignmentRow,
  type ShiftSwapRow,
} from "@/lib/graphql/scheduling";
import { workerOverviewQuery } from "@/lib/queries/worker";
import type { WorkerRecord, WorkerTabProps } from "@/routes/worker/use-worker-state";
import { useWorkerAreas } from "@/routes/worker/worker-areas";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { WithheldValue } from "@trenova/shared/components/withheld-value";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  daysUntil,
  formatRange,
  formatUnixDateMedium,
  getTodayDate,
} from "@trenova/shared/lib/date";
import {
  describeShiftPattern,
  formatShiftDate,
  formatShiftWindow,
  SWAP_STATUS_TONES,
} from "@trenova/shared/lib/scheduling";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { AvailabilityPreferenceValue } from "@trenova/shared/types/scheduling";
import { useState } from "react";
import { ConfirmDialog } from "@/components/confirm-dialog";
import { WORKER_FORM_PANES } from "../../worker-form-panes";
import { ListCapLine } from "../../list-more";
import { useWorkerInvalidation } from "../../use-worker-invalidation";
import { useWorkerFormGate } from "../../use-worker-form-gate";
import { TimeSkeleton } from "../time-skeleton";
import {
  availabilitySummary,
  longWeekday,
  shortWeekday,
  WEEK_ORDER,
  type StatedPreference,
} from "./schedule-model";
import { WeekPattern } from "./week-pattern";

const AssignShiftPane = lazyWorkPane(
  WORKER_FORM_PANES["assign-shift"],
  (module) => module.AssignShiftPane,
  { title: (props, t) => (props.moving ? t("Move to another shift") : t("Put on a shift")) },
);

type PreferenceChoice = AvailabilityPreferenceValue | "None";

type ScheduleViewProps = Pick<WorkerTabProps, "workerId" | "worker" | "open" | "onOpen">;

/**
 * One worker's side of the rota: the pattern they are on, what they have said about each
 * weekday, the swaps they are part of and when they were last home.
 */
export default function ScheduleView({ workerId, worker, open, onOpen }: ScheduleViewProps) {
  const t = useT();
  const pane = useWorkPane();
  const { allowed: canAssign } = usePermission(Resource.WorkerSchedule, Operation.Assign);
  const { allowed: canReadSwaps } = usePermission(Resource.ShiftSwap, Operation.Read);

  const assignmentsQuery = useQuery({
    queryKey: [SHIFT_ASSIGNMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerShiftAssignments(workerId, undefined, { signal }),
  });
  const preferencesQuery = useQuery({
    queryKey: [AVAILABILITY_PREFERENCES_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerAvailabilityPreferences(workerId, { signal }),
  });

  const invalidate = useWorkerInvalidation(workerId, "schedule");

  const opens = useWorkerFormGate(open.form, { "assign-shift": canAssign });

  if (assignmentsQuery.isPending) return <TimeSkeleton view="schedule" />;
  if (assignmentsQuery.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={assignmentsQuery.error}
        onRetry={() => void assignmentsQuery.refetch()}
        title={t("The schedule could not be loaded")}
      />
    );
  }

  // The server returns the newest 100 (SERVER_LIST_CAP), so the one in force (no end) is here.
  const assignments = assignmentsQuery.data;
  const current = assignments.find((row) => !row.effectiveTo) ?? null;
  const history = assignments.filter((row) => row.id !== current?.id);
  const preferences: StatedPreference[] = (preferencesQuery.data ?? []).map((row) => ({
    dayOfWeek: row.dayOfWeek,
    preference: row.preference as AvailabilityPreferenceValue,
  }));

  return (
    <div className="grid min-w-0 items-start gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
      <div className="flex min-w-0 flex-col gap-4">
        <ShiftPanel
          worker={worker}
          current={current}
          history={history}
          assignmentCount={assignments.length}
          canAssign={canAssign}
          onAssign={() =>
            pane.closeThen(() => onOpen({ tab: "time", view: "schedule", form: "assign-shift" }))
          }
          onEnded={invalidate}
        />
        <AvailabilityPanel
          workerId={workerId}
          preferences={preferences}
          loading={preferencesQuery.isPending}
          error={preferencesQuery.isError ? preferencesQuery.error : null}
          onRetry={() => void preferencesQuery.refetch()}
        />
      </div>
      <div className="flex min-w-0 flex-col gap-4">
        <HomeTimePanel workerId={workerId} worker={worker} />
        {canReadSwaps ? <SwapsPanel workerId={workerId} /> : null}
      </div>
      {canAssign ? (
        <AssignShiftPane.Component
          open={opens("assign-shift")}
          onOpenChange={(next) => {
            if (!next) onOpen({ tab: "time", view: "schedule" });
          }}
          workerId={workerId}
          moving={current !== null}
          preferences={preferences}
          onAssigned={invalidate}
        />
      ) : null}
    </div>
  );
}

function ShiftPanel({
  worker,
  current,
  history,
  assignmentCount,
  canAssign,
  onAssign,
  onEnded,
}: {
  worker: WorkerRecord;
  current: ShiftAssignmentRow | null;
  history: readonly ShiftAssignmentRow[];
  /** How many assignments the server returned; at its cap, older ones are not shown. */
  assignmentCount: number;
  canAssign: boolean;
  onAssign: () => void;
  onEnded: () => Promise<unknown>;
}) {
  const t = useT();
  const [confirmEnd, setConfirmEnd] = useState(false);
  const end = useApiMutation({
    resourceName: "Assignment",
    scope: { id: `worker-end-shift:${current?.id ?? "none"}` },
    mutationFn: (id: string) => endWorkerShiftAssignment(id, getTodayDate()),
    onSuccess: () => onEnded(),
  });
  const template = current?.shiftTemplate ?? null;
  const daysOfWeek = template?.daysOfWeek ?? null;
  const cycleWeeks = template?.cycleWeeks ?? null;
  const patternShown = template !== null && daysOfWeek !== null && cycleWeeks !== null;

  return (
    <SectionPanel title={t("Shift")}>
      <div className="flex flex-col items-start justify-between gap-4 p-3 sm:flex-row sm:items-center">
        <div className="max-w-[60ch] min-w-0">
          <p className="flex items-center gap-1.5 text-sm font-medium">
            {template?.color ? (
              <span
                aria-hidden
                className="size-2 shrink-0 rounded-full"
                style={{ backgroundColor: template.color }}
              />
            ) : null}
            {template?.name ?? t("Not on a shift")}
          </p>
          <p className="mt-0.5 text-xs text-muted-foreground">
            {template === null ? (
              worker.driverType === "OTR" ? (
                t(
                  "Over-the-road drivers are planned from hours of service and home time, not a shift pattern.",
                )
              ) : (
                t("They show on the rota with no rostered days until they are on a shift.")
              )
            ) : daysOfWeek === null || cycleWeeks === null ? (
              <span className="inline-flex items-center gap-1">
                <WithheldValue kind="shift_template" />
                <span aria-hidden>·</span>
                {formatShiftWindow(template.startMinute, template.durationMinutes)}
              </span>
            ) : (
              t(
                "{0} · {1}",
                describeShiftPattern(daysOfWeek, cycleWeeks),
                formatShiftWindow(template.startMinute, template.durationMinutes),
              )
            )}
          </p>
          {current ? (
            <p className="mt-0.5 text-xs text-muted-foreground tabular-nums">
              {cycleWeeks !== null && cycleWeeks > 1
                ? t(
                    "Since {0} · week {1} of the rotation",
                    formatShiftDate(current.effectiveFrom),
                    current.cycleOffsetWeeks + 1,
                  )
                : t("Since {0}", formatShiftDate(current.effectiveFrom))}
            </p>
          ) : null}
        </div>
        {canAssign ? (
          <div className="flex shrink-0 items-center gap-2">
            {current ? (
              <Button size="sm" variant="outline" onClick={() => setConfirmEnd(true)}>
                {t("End today")}
              </Button>
            ) : null}
            <Button
              size="sm"
              variant="secondary"
              {...preloadOn(AssignShiftPane)}
              onClick={onAssign}
            >
              {current ? t("Move to another shift") : t("Put on a shift")}
            </Button>
          </div>
        ) : null}
      </div>
      {current && patternShown ? (
        <WeekPattern
          daysOfWeek={daysOfWeek}
          startMinute={template.startMinute}
          durationMinutes={template.durationMinutes}
          className="px-3 pb-3"
        />
      ) : null}
      {history.length > 0 ? (
        <div className="border-t border-border-subtle">
          <p className="px-3 pt-2.5 pb-1 text-xs font-medium text-foreground-subtle">
            {t("Earlier shifts")}
          </p>
          <VirtualRows
            rows={history.map((row) => ({
              key: row.id,
              render: () => <PastAssignment assignment={row} />,
            }))}
            aria-label={t("Earlier shifts")}
            mask="card"
            className="max-h-44"
            contentClassName="px-1.5 pb-1.5"
            hoverGlide
            estimateSize={30}
            footer={<ListCapLine count={assignmentCount} />}
          />
        </div>
      ) : null}
      {current ? (
        <ConfirmDialog
          open={confirmEnd}
          onOpenChange={setConfirmEnd}
          title={t("End this shift today?")}
          description={t(
            "{0} stops being planned from {1} after today. They show on the rota with no rostered days until they are put on another shift.",
            worker.wholeName,
            template?.name ?? t("this shift"),
          )}
          confirmLabel={t("End shift")}
          pending={end.isPending}
          onConfirm={() => end.mutateAsync(current.id)}
        />
      ) : null}
    </SectionPanel>
  );
}

function PastAssignment({ assignment }: { assignment: ShiftAssignmentRow }) {
  const t = useT();
  return (
    <div
      data-glide-target
      className="flex items-center justify-between gap-3 rounded-md px-1.5 py-1.5 text-xs"
    >
      <span className="flex min-w-0 items-center gap-2">
        {assignment.shiftTemplate?.color ? (
          <span
            aria-hidden
            className="size-1.5 shrink-0 rounded-full"
            style={{ backgroundColor: assignment.shiftTemplate.color }}
          />
        ) : null}
        <span className="truncate">{assignment.shiftTemplate?.name ?? t("A shift")}</span>
      </span>
      <span className="shrink-0 font-mono text-muted-foreground tabular-nums">
        {assignment.effectiveTo
          ? formatRange(assignment.effectiveFrom, assignment.effectiveTo, {
              year: "always",
              join: "words",
              timezone: "UTC",
            })
          : formatShiftDate(assignment.effectiveFrom)}
      </span>
    </div>
  );
}

function AvailabilityPanel({
  workerId,
  preferences,
  loading,
  error,
  onRetry,
}: {
  workerId: string;
  preferences: readonly StatedPreference[];
  loading: boolean;
  error: Error | null;
  onRetry: () => void;
}) {
  const t = useT();
  const invalidate = useWorkerInvalidation(workerId, "schedule");
  const { allowed: canSet } = usePermission(Resource.WorkerSchedule, Operation.Update);
  const [changed, setChanged] = useState<{ day: number; tick: "a" | "b" } | null>(null);
  const save = useApiMutation({
    resourceName: "Availability",
    scope: { id: `worker-availability:${workerId}` },
    mutationFn: (input: { dayOfWeek: number; preference: AvailabilityPreferenceValue }) =>
      setWorkerAvailabilityPreference({ workerId, ...input }),
    onSuccess: (_result, input) => {
      setChanged((previous) => ({
        day: input.dayOfWeek,
        tick: previous?.tick === "a" ? "b" : "a",
      }));
      return invalidate();
    },
  });
  const items: { value: PreferenceChoice; label: string; disabled: boolean }[] = [
    { value: "Preferred", label: t("Prefer"), disabled: !canSet || save.isPending },
    { value: "Available", label: t("Can"), disabled: !canSet || save.isPending },
    { value: "Unavailable", label: t("Can't"), disabled: !canSet || save.isPending },
  ];

  return (
    <SectionPanel
      title={t("Stated availability")}
      hint={loading || error ? undefined : availabilitySummary(t, preferences, shortWeekday)}
    >
      {loading ? (
        <div aria-busy aria-label={t("Loading stated availability")} className="p-3">
          <div aria-hidden className="h-64 w-full ui-shimmer rounded-md" />
        </div>
      ) : error ? (
        <div className="p-3">
          <ErrorState
            layout="compact"
            error={error}
            onRetry={onRetry}
            title={t("The stated availability could not be loaded")}
          />
        </div>
      ) : (
        <>
          <ul className="flex flex-col py-1">
            {WEEK_ORDER.map((day) => {
              const stated = preferences.find((row) => row.dayOfWeek === day);
              const value: PreferenceChoice = stated?.preference ?? "None";
              return (
                <li
                  key={day}
                  data-changed={changed?.day === day ? changed.tick : undefined}
                  className="flex min-h-10 flex-wrap items-center justify-between gap-3 border-border-subtle px-3 py-1 not-first:border-t data-[changed=a]:animate-row-changed-a data-[changed=b]:animate-row-changed-b"
                >
                  <span className="flex items-center gap-2">
                    <span className="text-sm font-medium">{longWeekday(day)}</span>
                    {stated ? null : (
                      <span className="text-xs text-muted-foreground">{t("Nothing said")}</span>
                    )}
                  </span>
                  <SegmentedControl<PreferenceChoice>
                    aria-label={longWeekday(day)}
                    items={items}
                    value={value}
                    onValueChange={(next) => {
                      if (next === "None" || next === value) return;
                      void save
                        .mutateAsync({ dayOfWeek: day, preference: next })
                        .catch(() => undefined);
                    }}
                  />
                </li>
              );
            })}
          </ul>
          <p className="border-t border-border-subtle px-3 py-2.5 text-xs text-muted-foreground">
            {t(
              "A statement, never a constraint. Dispatch sees it when planning their next load and can override it; the rota shows where it did.",
            )}
          </p>
        </>
      )}
    </SectionPanel>
  );
}

function SwapsPanel({ workerId }: { workerId: string }) {
  const t = useT();
  const query = useQuery({
    queryKey: [SHIFT_SWAPS_KEY, workerId],
    queryFn: ({ signal }) => fetchShiftSwapRequests({ workerId }, { signal }),
  });
  const swaps = query.data ?? [];
  return (
    <SectionPanel title={t("Swaps")} count={swaps.length}>
      {query.isPending ? (
        <div aria-busy aria-label={t("Loading swaps")} className="p-3">
          <div aria-hidden className="h-10 w-full ui-shimmer rounded-md" />
        </div>
      ) : query.isError ? (
        <div className="p-3">
          <ErrorState
            layout="compact"
            error={query.error}
            onRetry={() => void query.refetch()}
            title={t("The swaps could not be loaded")}
          />
        </div>
      ) : swaps.length === 0 ? (
        <SectionPanelQuiet>{t("No swap requests.")}</SectionPanelQuiet>
      ) : (
        <VirtualRows
          rows={swaps.map((swap) => ({
            key: swap.id,
            render: () => <SwapRow swap={swap} workerId={workerId} />,
          }))}
          aria-label={t("Swaps")}
          mask="card"
          className="max-h-60"
          contentClassName="px-1.5 py-1"
          hoverGlide
          estimateSize={48}
        />
      )}
    </SectionPanel>
  );
}

function SwapRow({ swap, workerId }: { swap: ShiftSwapRow; workerId: string }) {
  const t = useT();
  const tone = SWAP_STATUS_TONES[swap.status] ?? SWAP_STATUS_TONES.Withdrawn;
  const outgoing = swap.requestingWorkerId === workerId;
  const other = outgoing ? swap.counterpartyWorker : swap.requestingWorker;
  const otherName = other ? `${other.firstName} ${other.lastName}` : null;
  return (
    <div
      data-glide-target
      className="flex items-center justify-between gap-2 rounded-md px-1.5 py-2 text-xs"
    >
      <span className="min-w-0">
        <span className="block truncate font-medium">
          {outgoing
            ? otherName
              ? t("Offered to {0}", otherName)
              : t("Offered")
            : otherName
              ? t("Offered by {0}", otherName)
              : t("Offered to them")}
        </span>
        <span className="block font-mono text-muted-foreground tabular-nums">
          {formatShiftDate(swap.shiftDate)}
        </span>
      </span>
      <Badge variant={tone.variant} dot>
        {t(tone.label)}
      </Badge>
    </div>
  );
}

function HomeTimePanel({ workerId, worker }: { workerId: string; worker: WorkerRecord }) {
  const t = useT();
  const areas = useWorkerAreas();
  const canReadTimeOff = areas.isDrawable({ tab: "time", view: "time-off" });
  const overview = useQuery(workerOverviewQuery(workerId));
  const next = overview.data?.nextTimeOff ?? null;
  const home = [worker.city, worker.state?.abbreviation].filter(Boolean).join(", ");

  return (
    <SectionPanel title={t("Home time")}>
      <div className="px-3 py-1">
        <DescriptionList layout="split">
          <DescriptionItem label={t("Last home")} numeric>
            {worker.lastHomeAt ? (
              <LastHome lastHomeAt={worker.lastHomeAt} />
            ) : (
              <span className="text-muted-foreground">{t("Not recorded")}</span>
            )}
          </DescriptionItem>
          {canReadTimeOff ? (
            <DescriptionItem label={t("Next time off")} numeric>
              {overview.isPending ? (
                <span aria-hidden className="inline-block h-3 w-24 ui-shimmer rounded-sm" />
              ) : overview.isError ? (
                <span className="text-muted-foreground">{t("Could not be loaded")}</span>
              ) : next ? (
                <>
                  {formatRange(next.startDate, next.endDate, { year: "always", join: "words" })}
                  {next.status === "Requested" ? (
                    <span className="text-muted-foreground"> {t("· waiting on a decision")}</span>
                  ) : null}
                </>
              ) : (
                <span className="text-muted-foreground">{t("None booked")}</span>
              )}
            </DescriptionItem>
          ) : null}
          <DescriptionItem label={t("Home")}>
            {home || <span className="text-muted-foreground">{t("Not recorded")}</span>}
          </DescriptionItem>
        </DescriptionList>
      </div>
    </SectionPanel>
  );
}

/** The day they were last home and how many days ago, moving at midnight. */
function LastHome({ lastHomeAt }: { lastHomeAt: number }) {
  const t = useT();
  const now = useNow("minute");
  const out = Math.max(0, -daysUntil(lastHomeAt, { now }));
  return (
    <>
      {formatUnixDateMedium(lastHomeAt)}
      <span className="text-muted-foreground">
        {" "}
        {t("· {0, plural, one {# day out} other {# days out}}", out)}
      </span>
    </>
  );
}
