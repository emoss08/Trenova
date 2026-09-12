import { useT } from "@trenova/shared/i18n/use-t";
import { FleetCodeAutocompleteField } from "@/components/autocomplete-fields";
import { useLocalStorage } from "@/hooks/use-local-storage";
import { usePermission } from "@/hooks/use-permission";
import { type ShiftTemplateRow } from "@/lib/graphql/scheduling";
import {
  isRotaDensity,
  matchesRotaSearch,
  ROTA_DENSITY_STORAGE_KEY,
  templateStats,
  type RotaDensity,
} from "@/lib/scheduling-board";
import { useQuery } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@trenova/shared/components/ui/tabs";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  addRotaWeeks,
  DAY_LABELS,
  dayMaskToDays,
  describeShiftPattern,
  formatShiftWindow,
  rotaStateTone,
  startOfRotaWeek,
  weeklyShiftMinutes,
} from "@trenova/shared/lib/scheduling";
import { formatHours } from "@trenova/shared/lib/timesheet";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import {
  CalendarRangeIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  ClockIcon,
  PlusIcon,
  RepeatIcon,
  Rows3Icon,
  Rows4Icon,
  SearchIcon,
  UsersIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { openSwapsQuery, rotaQuery, shiftTemplatesQuery } from "./queries";
import { RotaAttention } from "./rota-attention";
import { RotaBoard, rotaWeekLabel } from "./rota-board";
import { RotaEmpty, ShiftsEmpty } from "./scheduling-empty";
import { SchedulingOverview } from "./scheduling-overview";
import { RotaBoardSkeleton } from "./scheduling-skeleton";
import { ShiftTemplateDialog } from "./shift-template-dialog";
import { SwapQueue } from "./swap-queue";

type WeeksValue = "1" | "2" | "4";
type FilterValues = { fleetCodeId: string };
type TabValue = "rota" | "shifts" | "swaps";

const WEEK_ITEMS = [
  { value: "1", label: "Week" },
  { value: "2", label: "2 weeks" },
  { value: "4", label: "4 weeks" },
] satisfies { value: WeeksValue; label: string }[];

const DENSITY_ITEMS = [
  { value: "comfortable", label: "Comfortable", icon: Rows3Icon },
  { value: "compact", label: "Compact", icon: Rows4Icon },
] satisfies { value: RotaDensity; label: string; icon: typeof Rows3Icon }[];

const LEGEND_STATES = ["Scheduled", "Assigned", "TimeOff", "Leave", "Unavailable", "Off"] as const;
const EMPTY_TEMPLATES: ShiftTemplateRow[] = [];

export default function SchedulingConsole() {
  const t = useT();

  const { allowed: canReadRota } = usePermission(Resource.WorkerSchedule, Operation.Read);
  const { allowed: canReadShifts } = usePermission(Resource.ShiftTemplate, Operation.Read);
  const { allowed: canCreateShift } = usePermission(Resource.ShiftTemplate, Operation.Create);
  const { allowed: canUpdateShift } = usePermission(Resource.ShiftTemplate, Operation.Update);
  const { allowed: canReadSwaps } = usePermission(Resource.ShiftSwap, Operation.Read);

  const [tab, setTab] = useState<TabValue>(canReadRota ? "rota" : "shifts");
  const [weekStart, setWeekStart] = useState(() => startOfRotaWeek(Math.floor(Date.now() / 1000)));
  const [weeks, setWeeks] = useState<WeeksValue>("1");
  const [teamOnly, setTeamOnly] = useState(false);
  const [search, setSearch] = useState("");
  // Density is a reader's habit, not a property of the week, so it lives in
  // the browser rather than in the URL or on the server.
  const [storedDensity, setStoredDensity] = useLocalStorage<RotaDensity>(
    ROTA_DENSITY_STORAGE_KEY,
    "comfortable",
  );
  const density: RotaDensity = isRotaDensity(storedDensity) ? storedDensity : "comfortable";
  const [dialog, setDialog] = useState<{ template: ShiftTemplateRow | null } | null>(null);
  const today = getTodayDate();

  // The fleet filter is a form field so it is the same autocomplete the rest
  // of the product uses, with the same search and the same pop-out.
  const filterForm = useForm<FilterValues>({ defaultValues: { fleetCodeId: "" } });
  const fleetCodeId = useWatch({ control: filterForm.control, name: "fleetCodeId" });
  const boardFiltered = teamOnly || Boolean(fleetCodeId);
  const clearBoardFilters = () => {
    setTeamOnly(false);
    filterForm.reset();
  };

  const weekCount = Number(weeks);
  const rotaResult = useQuery({
    ...rotaQuery({ weekStart, weeks: weekCount, teamOnly, fleetCodeId: fleetCodeId || null }),
    enabled: canReadRota,
  });
  const templatesQuery = useQuery({ ...shiftTemplatesQuery(), enabled: canReadShifts });
  const swapsQuery = useQuery({ ...openSwapsQuery(), enabled: canReadSwaps });

  const rota = rotaResult.data;
  const visibleRows = useMemo(
    () => (rota?.rows ?? []).filter((row) => matchesRotaSearch(row, search)),
    [rota, search],
  );
  const templates = templatesQuery.data ?? EMPTY_TEMPLATES;
  const stats = useMemo(() => templateStats(templates), [templates]);
  const openSwaps = swapsQuery.data?.length ?? 0;
  const awaitingOffice = useMemo(
    () => (swapsQuery.data ?? []).filter((swap) => swap.status === "Accepted").length,
    [swapsQuery.data],
  );

  if (!canReadRota && !canReadShifts) return null;

  return (
    <div className="flex flex-col gap-4">
      {canReadRota ? (
        <SchedulingOverview
          rota={rota}
          swaps={swapsQuery.data}
          today={today}
          showSwaps={canReadSwaps}
        />
      ) : null}

      <Tabs value={tab} onValueChange={(value) => setTab(value as TabValue)}>
        <TabsList variant="underline">
          {canReadRota ? (
            <TabsTrigger value="rota">
              <CalendarRangeIcon className="size-3.5" />
              {t("Rota")}
            </TabsTrigger>
          ) : null}
          {canReadShifts ? (
            <TabsTrigger value="shifts">
              <ClockIcon className="size-3.5" />
              {t("Shifts")}
              {stats.active > 0 ? (
                <Badge variant="secondary" className="text-2xs ml-1.5 h-4 px-1 tabular-nums">
                  {stats.active}
                </Badge>
              ) : null}
            </TabsTrigger>
          ) : null}
          {canReadSwaps ? (
            <TabsTrigger value="swaps">
              <RepeatIcon className="size-3.5" />
              {t("Swaps")}
              {awaitingOffice > 0 ? (
                <Badge variant="warning" className="text-2xs ml-1.5 h-4 px-1 tabular-nums">
                  {awaitingOffice}
                </Badge>
              ) : openSwaps > 0 ? (
                <Badge variant="secondary" className="text-2xs ml-1.5 h-4 px-1 tabular-nums">
                  {openSwaps}
                </Badge>
              ) : null}
            </TabsTrigger>
          ) : null}
        </TabsList>

        {canReadRota ? (
          <TabsContent value="rota" className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-3">
              <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2">
                <div className="flex items-center gap-1">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => setWeekStart(addRotaWeeks(weekStart, -1))}
                    aria-label={t("Previous week")}
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
                    aria-label={t("Next week")}
                  >
                    <ChevronRightIcon className="size-3.5" />
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => setWeekStart(startOfRotaWeek(Math.floor(Date.now() / 1000)))}
                  >
                    {t("Today")}
                  </Button>
                </div>
                <Input
                  type="search"
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  placeholder={t("Find a name, shift or terminal")}
                  aria-label={t("Find on the board")}
                  leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
                  inputContainerClassName="min-w-48 flex-1"
                />
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <FormProvider {...filterForm}>
                  <div className="w-56 [&>div]:gap-0">
                    <FleetCodeAutocompleteField<FilterValues>
                      control={filterForm.control}
                      name="fleetCodeId"
                      placeholder={t("All fleets")}
                      clearable
                    />
                  </div>
                </FormProvider>
                <SegmentedControl<WeeksValue>
                  items={WEEK_ITEMS}
                  value={weeks}
                  onValueChange={setWeeks}
                  className="h-7"
                  aria-label={t("Weeks on the board")}
                />
                <SegmentedControl<RotaDensity>
                  items={DENSITY_ITEMS}
                  value={density}
                  className="h-7"
                  onValueChange={setStoredDensity}
                  aria-label={t("Board density")}
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
                    {t("My team")}
                  </TooltipTrigger>
                  <TooltipContent>{t("Only the people you answer for")}</TooltipContent>
                </Tooltip>
              </div>
            </div>

            {rotaResult.isLoading ? (
              <RotaBoardSkeleton density={density} weeks={weekCount} />
            ) : rota && rota.rows.length > 0 ? (
              <>
                <RotaAttention
                  rows={rota.rows}
                  swaps={swapsQuery.data}
                  onOpenSwaps={canReadSwaps ? () => setTab("swaps") : undefined}
                />
                {visibleRows.length === 0 ? (
                  <RotaEmpty
                    title={t("Nobody matches that")}
                    description={t("Nothing on the board fits the search. Clear it to see the whole week again.")}
                    onClearFilters={() => setSearch("")}
                  />
                ) : (
                  <RotaBoard rota={rota} rows={visibleRows} density={density} />
                )}
              </>
            ) : (
              <RotaEmpty
                title={t("Nobody on the board")}
                description={
                  boardFiltered
                    ? "Nobody in that team or fleet is on a shift this week. Clear the filter to see everyone, or put a worker on a shift from their Schedule tab."
                    : "Put a worker on a shift from their Schedule tab and their week appears here, composed from the pattern, their time off and the work already assigned."
                }
                onClearFilters={boardFiltered ? clearBoardFilters : undefined}
              />
            )}

            <div className="flex flex-wrap items-center gap-4 text-[11px]">
              {LEGEND_STATES.map((state) => {
                const tone = rotaStateTone(state);
                return (
                  <span key={state} className="flex items-center gap-1.5">
                    <span className={cn("size-2 rounded-full", tone.dot)} aria-hidden />
                    <span className="text-muted-foreground">{t(tone.label)}</span>
                  </span>
                );
              })}
              <span className="text-muted-foreground ml-auto">
                {t("A ringed cell is a conflict. The small number is loads dispatch already assigned. The cover row counts who can work each day.")}
              </span>
            </div>
          </TabsContent>
        ) : null}

        {canReadShifts ? (
          <TabsContent value="shifts" className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex flex-col gap-1">
                <p className="text-muted-foreground max-w-2xl text-xs">
                  {t("A pattern is a mask of working days plus a start and a length. An A/B pair is a single two-week shift and two assignments at different offsets — not two near-identical shifts.")}
                </p>
                {templatesQuery.data ? (
                  <p className="text-xs tabular-nums" aria-label={t("Pattern summary")}>
                    <span className="font-medium">{stats.active}</span>
                    <span className="text-muted-foreground"> {t("active ·")} </span>
                    <span className="font-medium">{stats.onPatterns}</span>
                    <span className="text-muted-foreground"> {t("people on a pattern")}</span>
                    {stats.averageWeeklyMinutes != null ? (
                      <>
                        <span className="text-muted-foreground"> · </span>
                        <span className="font-medium">
                          {formatHours(stats.averageWeeklyMinutes)}
                        </span>
                        <span className="text-muted-foreground"> {t("a week on average")}</span>
                      </>
                    ) : null}
                    {stats.retired > 0 ? (
                      <span className="text-muted-foreground"> {t("· {0} retired", stats.retired)}</span>
                    ) : null}
                  </p>
                ) : null}
              </div>
              {canCreateShift ? (
                <Button size="sm" onClick={() => setDialog({ template: null })}>
                  <PlusIcon className="size-3.5" />
                  {t("Add a shift")}
                </Button>
              ) : null}
            </div>

            {templatesQuery.isLoading ? (
              <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
                <Skeleton className="h-36 rounded-lg" />
                <Skeleton className="h-36 rounded-lg" />
                <Skeleton className="h-36 rounded-lg" />
              </div>
            ) : templates.length === 0 ? (
              <ShiftsEmpty
                title={t("No shifts yet")}
                description={t("A shift is a pattern of days and hours. Add one, then put workers on it from their Schedule tab and the board fills in.")}
                onCreate={canCreateShift ? () => setDialog({ template: null }) : undefined}
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
  const t = useT();

  const days = dayMaskToDays(template.daysOfWeek);
  const retired = template.status !== "Active";
  const weekMinutes = weeklyShiftMinutes(template.daysOfWeek, template.durationMinutes);

  return (
    <article
      aria-label={template.name}
      className={cn(
        "bg-card border-border/80 hover:border-border group flex flex-col gap-3 rounded-lg border p-3 transition-colors",
        retired && "opacity-70",
      )}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0">
          <p className="flex items-center gap-1.5 truncate text-sm font-medium">
            {template.color ? (
              <span
                className="size-2 shrink-0 rounded-full"
                style={{ backgroundColor: template.color }}
                aria-hidden
              />
            ) : null}
            {template.name}
          </p>
          <p className="text-muted-foreground text-xs tabular-nums">
            {template.code} · {describeShiftPattern(template.daysOfWeek, template.cycleWeeks)}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          {retired ? <Badge variant="inactive">{t("Retired")}</Badge> : null}
          <Badge variant={template.activeAssignmentCount > 0 ? "active" : "secondary"}>
            {t("{0} on it", template.activeAssignmentCount)}
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
                on ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground",
              )}
            >
              {label[0]}
            </span>
          );
        })}
      </div>

      <div className="flex items-center justify-between text-xs">
        <span className="text-muted-foreground tabular-nums">
          {formatShiftWindow(template.startMinute, template.durationMinutes)}
          <span className="text-foreground"> {t("· {0} a week", formatHours(weekMinutes))}</span>
        </span>
        {onEdit ? (
          <Button
            size="xs"
            variant="ghost"
            className="opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
            onClick={onEdit}
            aria-label={`Edit ${template.name}`}
          >
            {t("Edit")}
          </Button>
        ) : null}
      </div>
    </article>
  );
}
