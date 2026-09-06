import { usePermission } from "@/hooks/use-permission";
import {
  AVAILABILITY_PREFERENCES_KEY,
  endWorkerShiftAssignment,
  fetchShiftSwapRequests,
  fetchWorkerAvailabilityPreferences,
  fetchWorkerShiftAssignments,
  ROTA_KEY,
  setWorkerAvailabilityPreference,
  SHIFT_ASSIGNMENTS_KEY,
  SHIFT_SWAPS_KEY,
  type ShiftAssignmentRow,
} from "@/lib/graphql/scheduling";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  AVAILABILITY_TONES,
  DAY_LABELS,
  DAY_LABELS_LONG,
  dayMaskToDays,
  describeShiftPattern,
  formatShiftDate,
  formatShiftWindow,
  SWAP_STATUS_TONES,
} from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { AvailabilityPreferenceValue } from "@trenova/shared/types/scheduling";
import { CalendarClockIcon, PlusIcon, RepeatIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { AssignShiftDialog } from "./scheduling/assign-shift-dialog";

type PreferenceChoice = AvailabilityPreferenceValue | "None";

const PREFERENCE_ITEMS = [
  { value: "Preferred", label: "Prefer" },
  { value: "Available", label: "Can" },
  { value: "Unavailable", label: "Can't" },
] satisfies { value: PreferenceChoice; label: string }[];

/**
 * One worker's side of the rota: the pattern they are on, what they have said
 * about each weekday, and the swaps they are part of.
 */
export default function WorkerScheduleTab({ workerId }: { workerId: string }) {
  const queryClient = useQueryClient();
  const { allowed: canRead } = usePermission(Resource.WorkerSchedule, Operation.Read);
  const { allowed: canAssign } = usePermission(Resource.WorkerSchedule, Operation.Assign);
  const { allowed: canSetPreference } = usePermission(Resource.WorkerSchedule, Operation.Update);
  const { allowed: canReadSwaps } = usePermission(Resource.ShiftSwap, Operation.Read);
  const [assignOpen, setAssignOpen] = useState(false);

  const assignmentsQuery = useQuery({
    queryKey: [SHIFT_ASSIGNMENTS_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerShiftAssignments(workerId, undefined, { signal }),
    enabled: canRead && Boolean(workerId),
  });
  const preferencesQuery = useQuery({
    queryKey: [AVAILABILITY_PREFERENCES_KEY, workerId],
    queryFn: ({ signal }) => fetchWorkerAvailabilityPreferences(workerId, { signal }),
    enabled: canRead && Boolean(workerId),
  });
  const swapsQuery = useQuery({
    queryKey: [SHIFT_SWAPS_KEY, workerId],
    queryFn: ({ signal }) => fetchShiftSwapRequests({ workerId }, { signal }),
    enabled: canReadSwaps && Boolean(workerId),
  });

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: [SHIFT_ASSIGNMENTS_KEY, workerId] });
    void queryClient.invalidateQueries({ queryKey: [AVAILABILITY_PREFERENCES_KEY, workerId] });
    void queryClient.invalidateQueries({ queryKey: [ROTA_KEY] });
  };

  const { mutate: savePreference, isPending: savingPreference } = useMutation({
    mutationFn: (input: { dayOfWeek: number; preference: AvailabilityPreferenceValue }) =>
      setWorkerAvailabilityPreference({ workerId, ...input }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [AVAILABILITY_PREFERENCES_KEY, workerId] });
      void queryClient.invalidateQueries({ queryKey: [ROTA_KEY] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const { mutate: endAssignment, isPending: endingAssignment } = useMutation({
    mutationFn: (id: string) => endWorkerShiftAssignment(id, getTodayDate()),
    onSuccess: () => {
      toast.success("Assignment ended");
      invalidate();
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (!canRead) return null;
  if (assignmentsQuery.isLoading) {
    return (
      <div className="flex flex-col gap-4">
        <Skeleton className="h-32 w-full rounded-lg" />
        <Skeleton className="h-64 w-full rounded-lg" />
      </div>
    );
  }

  const assignments = assignmentsQuery.data ?? [];
  const current = assignments.find((row) => !row.effectiveTo);
  const history = assignments.filter((row) => row.id !== current?.id);
  const preferences = preferencesQuery.data ?? [];
  const swaps = swapsQuery.data ?? [];
  const currentDays = current?.shiftTemplate ? dayMaskToDays(current.shiftTemplate.daysOfWeek) : [];

  return (
    <div className="flex flex-col gap-4">
      <section className="rounded-lg border p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex items-start gap-3">
            <span className="bg-accent inline-flex size-7 shrink-0 items-center justify-center rounded-md">
              <CalendarClockIcon className="size-4" />
            </span>
            <div className="min-w-0">
              <h3 className="flex items-center gap-1.5 text-sm font-semibold">
                {current?.shiftTemplate?.color ? (
                  <span
                    className="size-2 shrink-0 rounded-full"
                    style={{ backgroundColor: current.shiftTemplate.color }}
                    aria-hidden
                  />
                ) : null}
                {current?.shiftTemplate?.name ?? "Not on a shift"}
              </h3>
              <p className="text-muted-foreground text-xs">
                {current?.shiftTemplate
                  ? `${describeShiftPattern(current.shiftTemplate.daysOfWeek, current.shiftTemplate.cycleWeeks)} · ${formatShiftWindow(current.shiftTemplate.startMinute, current.shiftTemplate.durationMinutes)}`
                  : "They show on the rota with no rostered days until they are."}
              </p>
              {current ? (
                <p className="text-muted-foreground mt-1 text-xs tabular-nums">
                  Since {formatShiftDate(current.effectiveFrom)}
                  {current.shiftTemplate && current.shiftTemplate.cycleWeeks > 1
                    ? ` · week ${current.cycleOffsetWeeks + 1} of the rotation`
                    : ""}
                </p>
              ) : null}
            </div>
          </div>
          {canAssign ? (
            <div className="flex items-center gap-1.5">
              {current ? (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={endingAssignment}
                  onClick={() => endAssignment(current.id)}
                >
                  End today
                </Button>
              ) : null}
              <Button size="sm" onClick={() => setAssignOpen(true)}>
                <PlusIcon className="size-3.5" />
                {current ? "Move to another shift" : "Put on a shift"}
              </Button>
            </div>
          ) : null}
        </div>

        {current ? (
          <div className="mt-3 flex items-center gap-1">
            {DAY_LABELS.map((label, index) => {
              const on = currentDays.includes(index);
              return (
                <span
                  key={label}
                  className={cn(
                    "grid h-7 flex-1 place-items-center rounded-md text-[11px] font-medium",
                    on ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground",
                  )}
                >
                  {label}
                </span>
              );
            })}
          </div>
        ) : null}

        {history.length > 0 ? (
          <ul className="mt-3 flex flex-col gap-1 border-t pt-3 text-xs">
            {history.map((row) => (
              <PastAssignment key={row.id} assignment={row} />
            ))}
          </ul>
        ) : null}
      </section>

      <section className="rounded-lg border p-4">
        <h3 className="text-sm font-semibold">Stated availability</h3>
        <p className="text-muted-foreground text-xs">
          A statement, never a constraint. Dispatch can override it, and the rota shows where it did
          rather than hiding the override.
        </p>

        {preferencesQuery.isLoading ? (
          <Skeleton className="mt-3 h-40 w-full" />
        ) : (
          <ul className="mt-3 flex flex-col gap-1.5">
            {DAY_LABELS_LONG.map((label, dayOfWeek) => {
              const stated = preferences.find((row) => row.dayOfWeek === dayOfWeek);
              const value = (stated?.preference ?? "None") as PreferenceChoice;
              const tone = stated ? AVAILABILITY_TONES[stated.preference] : null;
              return (
                <li
                  key={label}
                  className="flex flex-wrap items-center justify-between gap-2 border-t py-1.5 text-xs first:border-t-0"
                >
                  <span className="flex items-center gap-2">
                    <span className="w-24 font-medium">{label}</span>
                    {tone ? (
                      <Badge variant={tone.variant}>{tone.label}</Badge>
                    ) : (
                      <span className="text-muted-foreground">Nothing said</span>
                    )}
                  </span>
                  <SegmentedControl<PreferenceChoice>
                    items={PREFERENCE_ITEMS.map((item) => ({
                      ...item,
                      disabled: !canSetPreference || savingPreference,
                    }))}
                    value={value}
                    onValueChange={(next) => {
                      if (next === "None") return;
                      savePreference({ dayOfWeek, preference: next });
                    }}
                    aria-label={`${label} availability`}
                  />
                </li>
              );
            })}
          </ul>
        )}
      </section>

      {canReadSwaps ? (
        <section className="rounded-lg border p-4">
          <div className="flex items-center gap-2">
            <RepeatIcon className="text-muted-foreground size-4" />
            <h3 className="text-sm font-semibold">Swaps</h3>
          </div>
          {swaps.length === 0 ? (
            <p className="text-muted-foreground mt-2 text-xs">No swap requests.</p>
          ) : (
            <ul className="mt-2 flex flex-col gap-1 text-xs">
              {swaps.map((swap) => {
                const tone = SWAP_STATUS_TONES[swap.status] ?? SWAP_STATUS_TONES.Withdrawn;
                const outgoing = swap.requestingWorkerId === workerId;
                return (
                  <li
                    key={swap.id}
                    className="flex flex-wrap items-center justify-between gap-2 border-t py-1.5 first:border-t-0"
                  >
                    <span className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">
                        {outgoing ? "Offered" : "Offered to them"}
                      </span>
                      <span className="text-muted-foreground tabular-nums">
                        {formatShiftDate(swap.shiftDate)}
                      </span>
                    </span>
                    <Badge variant={tone.variant}>{tone.label}</Badge>
                  </li>
                );
              })}
            </ul>
          )}
        </section>
      ) : null}

      <AssignShiftDialog
        open={assignOpen}
        onOpenChange={setAssignOpen}
        workerId={workerId}
        onAssigned={invalidate}
      />
    </div>
  );
}

function PastAssignment({ assignment }: { assignment: ShiftAssignmentRow }) {
  return (
    <li className="text-muted-foreground flex flex-wrap items-center justify-between gap-2">
      <span className="flex items-center gap-2">
        {assignment.shiftTemplate?.color ? (
          <span
            className="size-1.5 rounded-full"
            style={{ backgroundColor: assignment.shiftTemplate.color }}
            aria-hidden
          />
        ) : null}
        {assignment.shiftTemplate?.name ?? "Shift"}
      </span>
      <span className="tabular-nums">
        {formatShiftDate(assignment.effectiveFrom)}
        {assignment.effectiveTo ? ` – ${formatShiftDate(assignment.effectiveTo)}` : ""}
      </span>
    </li>
  );
}
