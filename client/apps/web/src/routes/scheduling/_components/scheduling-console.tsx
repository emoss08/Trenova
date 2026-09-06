import { FleetCodeAutocompleteField } from "@/components/autocomplete-fields";
import { EmptyState } from "@/components/empty-state";
import { KpiStat } from "@/components/kpi/kpi-stat";
import { usePermission } from "@/hooks/use-permission";
import {
  fetchRota,
  fetchShiftSwapRequests,
  fetchShiftTemplates,
  ROTA_KEY,
  SHIFT_SWAPS_KEY,
  SHIFT_TEMPLATES_KEY,
  type ShiftTemplateRow,
} from "@/lib/graphql/scheduling";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import {
  addRotaWeeks,
  dayMaskToDays,
  DAY_LABELS,
  describeShiftPattern,
  formatShiftWindow,
  rotaStateTone,
  startOfRotaWeek,
} from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  AlertTriangleIcon,
  CalendarDaysIcon,
  CalendarRangeIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClockIcon,
  PlusIcon,
  RepeatIcon,
  UserRoundXIcon,
  UsersIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { RotaBoard, rotaWeekLabel } from "./rota-board";
import { ShiftTemplateDialog } from "./shift-template-dialog";
import { SwapQueue } from "./swap-queue";

type WeeksValue = "1" | "2" | "4";
type FilterValues = { fleetCodeId: string };

const WEEK_ITEMS = [
  { value: "1", label: "Week" },
  { value: "2", label: "2 weeks" },
  { value: "4", label: "4 weeks" },
] satisfies { value: WeeksValue; label: string }[];

const LEGEND_STATES = ["Scheduled", "Assigned", "TimeOff", "Leave", "Unavailable", "Off"] as const;

export default function SchedulingConsole() {
  const { allowed: canReadRota } = usePermission(Resource.WorkerSchedule, Operation.Read);
  const { allowed: canReadShifts } = usePermission(Resource.ShiftTemplate, Operation.Read);
  const { allowed: canCreateShift } = usePermission(Resource.ShiftTemplate, Operation.Create);
  const { allowed: canUpdateShift } = usePermission(Resource.ShiftTemplate, Operation.Update);
  const { allowed: canReadSwaps } = usePermission(Resource.ShiftSwap, Operation.Read);

  const [weekStart, setWeekStart] = useState(() => startOfRotaWeek(Math.floor(Date.now() / 1000)));
  const [weeks, setWeeks] = useState<WeeksValue>("1");
  const [teamOnly, setTeamOnly] = useState(false);
  const [dialog, setDialog] = useState<{ template: ShiftTemplateRow | null } | null>(null);

  // The fleet filter is a form field so it is the same autocomplete the rest
  // of the product uses, with the same search and the same pop-out.
  const filterForm = useForm<FilterValues>({ defaultValues: { fleetCodeId: "" } });
  const fleetCodeId = useWatch({ control: filterForm.control, name: "fleetCodeId" });

  const weekCount = Number(weeks);
  const rotaQuery = useQuery({
    queryKey: [ROTA_KEY, weekStart, weekCount, teamOnly, fleetCodeId],
    queryFn: ({ signal }) =>
      fetchRota(
        {
          at: weekStart,
          weeks: weekCount,
          teamOnly,
          fleetCodeId: fleetCodeId || null,
        },
        { signal },
      ),
    enabled: canReadRota,
  });
  const templatesQuery = useQuery({
    queryKey: [SHIFT_TEMPLATES_KEY],
    queryFn: ({ signal }) => fetchShiftTemplates(undefined, { signal }),
    enabled: canReadShifts,
  });
  const swapsQuery = useQuery({
    queryKey: [SHIFT_SWAPS_KEY, "open"],
    queryFn: ({ signal }) => fetchShiftSwapRequests({ openOnly: true }, { signal }),
    enabled: canReadSwaps,
  });

  const openSwaps = swapsQuery.data?.length ?? 0;
  const rota = rotaQuery.data;
  const coverage = useMemo(() => {
    if (!rota) return { people: 0, days: 0, conflicts: 0, unrostered: 0 };
    return {
      people: rota.rows.length,
      days: rota.scheduledDays,
      conflicts: rota.conflicts,
      unrostered: rota.rows.filter((row) => row.scheduledDays === 0).length,
    };
  }, [rota]);

  if (!canReadRota && !canReadShifts) return null;

  const templates = templatesQuery.data ?? [];

  return (
    <div className="flex flex-col gap-4">
      <Tabs defaultValue={canReadRota ? "rota" : "shifts"}>
        <TabsList>
          {canReadRota ? (
            <TabsTrigger value="rota">
              <CalendarRangeIcon className="size-3.5" />
              Rota
            </TabsTrigger>
          ) : null}
          {canReadShifts ? (
            <TabsTrigger value="shifts">
              <ClockIcon className="size-3.5" />
              Shifts
              {templates.length > 0 ? (
                <Badge variant="secondary" className="ml-1.5">
                  {templates.length}
                </Badge>
              ) : null}
            </TabsTrigger>
          ) : null}
          {canReadSwaps ? (
            <TabsTrigger value="swaps">
              <RepeatIcon className="size-3.5" />
              Swaps
              {openSwaps > 0 ? (
                <Badge variant="active" className="ml-1.5">
                  {openSwaps}
                </Badge>
              ) : null}
            </TabsTrigger>
          ) : null}
        </TabsList>

        {canReadRota ? (
          <TabsContent value="rota" className="flex flex-col gap-4">
            <div className="grid grid-cols-8 gap-3">
              <KpiStat
                label="On the board"
                value={String(coverage.people)}
                icon={<UsersIcon className="size-[11px]" />}
                sub="Active workers in the filter"
              />
              <KpiStat
                label="Days rostered"
                value={String(coverage.days)}
                icon={<CalendarDaysIcon className="size-[11px]" />}
                sub={`Across ${weekCount === 1 ? "the week" : `${weekCount} weeks`}`}
              />
              <KpiStat
                label="Conflicts"
                value={String(coverage.conflicts)}
                tone={coverage.conflicts > 0 ? "danger" : "success"}
                icon={<AlertTriangleIcon className="size-[11px]" />}
                sub="Rostered on a day they cannot work"
              />
              <KpiStat
                label="No shift"
                value={String(coverage.unrostered)}
                tone={coverage.unrostered > 0 ? "warning" : "muted"}
                icon={<UserRoundXIcon className="size-[11px]" />}
                sub="On the roster, on no pattern"
              />
            </div>

            <div className="flex flex-wrap items-end justify-between gap-3">
              <div className="flex items-center gap-1">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setWeekStart(addRotaWeeks(weekStart, -1))}
                  aria-label="Previous week"
                >
                  <ChevronLeftIcon className="size-3.5" />
                </Button>
                <span className="min-w-48 text-center text-sm font-medium tabular-nums">
                  {rotaWeekLabel(weekStart, weekCount)}
                </span>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setWeekStart(addRotaWeeks(weekStart, 1))}
                  aria-label="Next week"
                >
                  <ChevronRightIcon className="size-3.5" />
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setWeekStart(startOfRotaWeek(Math.floor(Date.now() / 1000)))}
                >
                  Today
                </Button>
              </div>

              <div className="flex flex-wrap items-end gap-2">
                <FormProvider {...filterForm}>
                  <div className="w-56">
                    <FleetCodeAutocompleteField<FilterValues>
                      control={filterForm.control}
                      name="fleetCodeId"
                      placeholder="All fleets"
                      clearable
                    />
                  </div>
                </FormProvider>
                <SegmentedControl<WeeksValue>
                  items={WEEK_ITEMS}
                  value={weeks}
                  onValueChange={setWeeks}
                  aria-label="Weeks on the board"
                />
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <Button
                        size="sm"
                        variant={teamOnly ? "default" : "outline"}
                        onClick={() => setTeamOnly((value) => !value)}
                        aria-pressed={teamOnly}
                      />
                    }
                  >
                    <UsersIcon className="size-3.5" />
                    My team
                  </TooltipTrigger>
                  <TooltipContent>Only the people you answer for</TooltipContent>
                </Tooltip>
              </div>
            </div>

            {rotaQuery.isLoading ? (
              <Skeleton className="h-72 w-full rounded-xl" />
            ) : rota && rota.rows.length > 0 ? (
              <RotaBoard rota={rota} />
            ) : (
              <EmptyState
                className="max-w-none"
                title="Nobody on the board"
                description={
                  teamOnly || fleetCodeId
                    ? "Nobody matches the filter for this week. Widen it, or put a worker on a shift from their Schedule tab."
                    : "Put a worker on a shift from their Schedule tab and their week appears here."
                }
                icons={[CalendarRangeIcon, UsersIcon, ClockIcon]}
              />
            )}

            <div className="flex flex-wrap items-center gap-4 text-[11px]">
              {LEGEND_STATES.map((state) => {
                const tone = rotaStateTone(state);
                return (
                  <span key={state} className="flex items-center gap-1.5">
                    <span className={cn("size-2 rounded-full", tone.dot)} aria-hidden />
                    <span className="text-muted-foreground">{tone.label}</span>
                  </span>
                );
              })}
              <span className="text-muted-foreground ml-auto">
                A ringed cell is a conflict. The small number is loads dispatch already assigned.
              </span>
            </div>
          </TabsContent>
        ) : null}

        {canReadShifts ? (
          <TabsContent value="shifts" className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <p className="text-muted-foreground max-w-2xl text-xs">
                A pattern is a mask of working days plus a start and a length. An A/B pair is a
                single two-week shift and two assignments at different offsets — not two
                near-identical shifts.
              </p>
              {canCreateShift ? (
                <Button size="sm" onClick={() => setDialog({ template: null })}>
                  <PlusIcon className="size-3.5" />
                  Add a shift
                </Button>
              ) : null}
            </div>

            {templatesQuery.isLoading ? (
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                <Skeleton className="h-36 rounded-xl" />
                <Skeleton className="h-36 rounded-xl" />
                <Skeleton className="h-36 rounded-xl" />
              </div>
            ) : templates.length === 0 ? (
              <EmptyState
                className="max-w-none"
                title="No shifts yet"
                description="Add a pattern, then put workers on it from their Schedule tab."
                icons={[ClockIcon, CalendarDaysIcon, RepeatIcon]}
                action={
                  canCreateShift
                    ? {
                        label: "Add a shift",
                        icon: PlusIcon,
                        onClick: () => setDialog({ template: null }),
                      }
                    : undefined
                }
              />
            ) : (
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                {templates.map((template) => (
                  <ShiftCard
                    key={template.id}
                    template={template}
                    onEdit={canUpdateShift ? () => setDialog({ template }) : undefined}
                  />
                ))}
              </div>
            )}
          </TabsContent>
        ) : null}

        {canReadSwaps ? (
          <TabsContent value="swaps">
            <SwapQueue />
          </TabsContent>
        ) : null}
      </Tabs>

      <ShiftTemplateDialog
        open={dialog !== null}
        onOpenChange={(open) => !open && setDialog(null)}
        template={dialog?.template ?? null}
      />
    </div>
  );
}

function ShiftCard({ template, onEdit }: { template: ShiftTemplateRow; onEdit?: () => void }) {
  const days = dayMaskToDays(template.daysOfWeek);
  const retired = template.status !== "Active";
  const accent = template.color || "var(--primary)";

  return (
    <div
      className={cn(
        "border-border/80 bg-card hover:border-border group relative flex flex-col gap-3 overflow-hidden rounded-xl border p-4 transition-colors",
        retired && "opacity-70",
      )}
    >
      <span
        className="absolute inset-y-0 left-0 w-1"
        style={{ backgroundColor: accent }}
        aria-hidden
      />
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium">{template.name}</p>
          <p className="text-muted-foreground text-xs tabular-nums">
            {template.code} · {describeShiftPattern(template.daysOfWeek, template.cycleWeeks)}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          {retired ? <Badge variant="inactive">Retired</Badge> : null}
          <Badge variant={template.activeAssignmentCount > 0 ? "active" : "secondary"}>
            {template.activeAssignmentCount} on it
          </Badge>
        </div>
      </div>

      <div className="flex items-center gap-1">
        {DAY_LABELS.map((label, index) => {
          const on = days.includes(index);
          return (
            <span
              key={label}
              className={cn(
                "grid h-6 flex-1 place-items-center rounded-md text-[10px] font-medium",
                on ? "text-white" : "bg-muted text-muted-foreground",
              )}
              style={on ? { backgroundColor: accent } : undefined}
            >
              {label[0]}
            </span>
          );
        })}
      </div>

      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground tabular-nums">
          {formatShiftWindow(template.startMinute, template.durationMinutes)}
        </span>
        {onEdit ? (
          <Button
            size="xs"
            variant="ghost"
            className="opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            onClick={onEdit}
            aria-label={`Edit ${template.name}`}
          >
            Edit
          </Button>
        ) : null}
      </div>
    </div>
  );
}
